package experiments

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	parser "github.com/Netflix/p2plab/cue/parser"
	"github.com/Netflix/p2plab/metadata"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestExperimentDefinition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db, cleanup := newTestDB(t, "exptestdir")
	defer func() {
		require.NoError(t, cleanup())
	}()
	var ids []string
	t.Run("Experiment Creation And Retrieval", func(t *testing.T) {
		sourceFiles := []string{
			"../cue/cue.mod/p2plab_example1.cue",
			"../cue/cue.mod/p2plab_example2.cue",
		}
		for _, sourceFile := range sourceFiles {
			name := strings.Split(sourceFile, "/")
			exp1 := newTestExperiment(t, sourceFile, name[len(name)-1])
			ids = append(ids, exp1.ID)
			exp2, err := db.CreateExperiment(ctx, exp1)
			require.NoError(t, err)
			require.Equal(t, exp1.ID, exp2.ID)
			require.Equal(t, exp1.Status, exp2.Status)
			require.Equal(t, exp1.Definition, exp2.Definition)
			exp3, err := db.GetExperiment(ctx, exp1.ID)
			require.NoError(t, err)
			require.Equal(t, exp1.ID, exp3.ID)
			require.Equal(t, exp1.Status, exp3.Status)
			require.Equal(t, exp1.Definition, exp3.Definition)
		}
	})
	t.Run("List Experiments", func(t *testing.T) {
		experiments, err := db.ListExperiments(ctx)
		require.NoError(t, err)
		for _, experiment := range experiments {
			if experiment.ID != ids[0] && experiment.ID != ids[1] {
				t.Error("bad experiment id found")
			}
		}
	})
	t.Run("Update Experiments", func(t *testing.T) {
		for _, id := range ids {
			exp, err := db.GetExperiment(ctx, id)
			require.NoError(t, err)
			prevUpdateAt := exp.UpdatedAt
			exp.Labels = append(exp.Labels, "test label")
			exp, err = db.UpdateExperiment(ctx, exp)
			require.NoError(t, err)
			require.True(t, exp.UpdatedAt.After(prevUpdateAt))
		}
	})
	t.Run("Label Experiments", func(t *testing.T) {
		exps, err := db.LabelExperiments(
			ctx,
			ids,
			[]string{"should be present"},
			[]string{"test label"},
		)
		require.NoError(t, err)
		for _, exp := range exps {
			require.Len(t, exp.Labels, 1)
			require.Equal(t, exp.Labels[0], "should be present")
		}
	})
	t.Run("Delete Experiment", func(t *testing.T) {
		require.NoError(t, db.DeleteExperiment(ctx, ids[0]))
		_, err := db.GetExperiment(ctx, ids[0])
		require.Error(t, err)
		_, err = db.GetExperiment(ctx, ids[1])
		require.NoError(t, err)
	})
}

func newTestExperiment(t *testing.T, sourceFile, name string) metadata.Experiment {
	edef, err := Parse(sourceFile)
	require.NoError(t, err)
	return metadata.Experiment{
		ID:         uuid.New().String(),
		Status:     metadata.ExperimentRunning,
		Definition: edef,
	}
}

func newTestDB(t *testing.T, path string) (metadata.DB, func() error) {
	db, err := metadata.NewDB(path)
	require.NoError(t, err)
	cleanup := func() error {
		if err := db.Close(); err != nil {
			return err
		}
		return os.RemoveAll(path)
	}
	return db, cleanup
}
