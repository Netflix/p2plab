package helpers

import (
	"context"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/labd/controlapi"
	"github.com/Netflix/p2plab/metadata"
	"github.com/Netflix/p2plab/nodes"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	bolt "go.etcd.io/bbolt"
)

// TODO(bonedaddy): not sure if this is the best way to go about sharing code between routers

// Helper abstracts commonly used functions to be shared by any router
type Helper struct {
	db       metadata.DB
	provider p2plab.NodeProvider
	client   *httputil.Client
}

// New instantiates our helper type
func New(db metadata.DB, provider p2plab.NodeProvider, client *httputil.Client) *Helper {
	return &Helper{db, provider, client}
}

// CreateCluster enables creating the nodes in a cluster, waiting for them to be healthy before returning
func (h *Helper) CreateCluster(ctx context.Context, cdef metadata.ClusterDefinition, name string) (metadata.Cluster, error) {
	var (
		cluster = metadata.Cluster{
			ID:         name,
			Status:     metadata.ClusterCreating,
			Definition: cdef,
			Labels: append([]string{
				name,
			}, cdef.GenerateLabels()...),
		}
		err error
	)

	cluster, err = h.db.CreateCluster(ctx, cluster)
	if err != nil {
		return cluster, err
	}

	zerolog.Ctx(ctx).Info().Str("cid", name).Msg("Creating node group")
	ng, err := h.provider.CreateNodeGroup(ctx, name, cdef)
	if err != nil {
		return cluster, err
	}

	var mns []metadata.Node
	cluster.Status = metadata.ClusterConnecting
	if err := h.db.Update(ctx, func(tx *bolt.Tx) error {
		var err error
		tctx := metadata.WithTransactionContext(ctx, tx)
		cluster, err = h.db.UpdateCluster(tctx, cluster)
		if err != nil {
			return err
		}

		mns, err = h.db.CreateNodes(tctx, cluster.ID, ng.Nodes)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return cluster, err
	}

	var ns = make([]p2plab.Node, len(mns))
	for i, n := range mns {
		ns[i] = controlapi.NewNode(h.client, n)
	}

	if err := nodes.WaitHealthy(ctx, ns); err != nil {
		return cluster, err
	}

	cluster.Status = metadata.ClusterCreated
	return h.db.UpdateCluster(ctx, cluster)
}

func (h *Helper) DeleteCluster(ctx context.Context, name string) error {
	logger := zerolog.Ctx(ctx).With().Str("name", name).Logger()
	ctx = logger.WithContext(ctx)

	cluster, err := h.db.GetCluster(ctx, name)
	if err != nil {
		return errors.Wrapf(err, "failed to get cluster %q", name)
	}

	if cluster.Status != metadata.ClusterDestroying {
		cluster.Status = metadata.ClusterDestroying
		cluster, err = h.db.UpdateCluster(ctx, cluster)
		if err != nil {
			return errors.Wrap(err, "failed to update cluster status to destroying")
		}
	}

	ns, err := h.db.ListNodes(ctx, cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to list nodes")
	}

	ng := &p2plab.NodeGroup{
		ID:    cluster.ID,
		Nodes: ns,
	}

	logger.Info().Msg("Destroying node group")
	err = h.provider.DestroyNodeGroup(ctx, ng)
	if err != nil {
		return errors.Wrap(err, "failed to destroy node group")
	}

	logger.Info().Msg("Deleting cluster metadata")
	err = h.db.DeleteCluster(ctx, cluster.ID)
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster metadata")
	}

	logger.Info().Msg("Destroyed cluster")
	return nil
}
