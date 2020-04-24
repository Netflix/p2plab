package experiments

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	parser "github.com/Netflix/p2plab/cue/parser"
	"github.com/Netflix/p2plab/metadata"
	"github.com/google/uuid"
)

func TestExperimentDefinition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db, cleanup := newTestDB(t, "exptestdir")
	defer func() {
		if err := cleanup(); err != nil {
			t.Fatal(err)
		}
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
			if err != nil {
				t.Fatal(err)
			}
			if exp1.ID != exp2.ID {
				t.Fatal("bad id")
			}
			if exp1.Status != exp2.Status {
				t.Fatal("bad status")
			}
			if !reflect.DeepEqual(exp1.Definition, exp2.Definition) {
				t.Fatal("bad definition")
			}
			exp3, err := db.GetExperiment(ctx, exp1.ID)
			if err != nil {
				t.Fatal(err)
			}
			if exp1.ID != exp3.ID {
				t.Fatal("bad id")
			}
			if exp1.Status != exp3.Status {
				t.Fatal("bad status")
			}
			if !reflect.DeepEqual(exp1.Definition, exp3.Definition) {
				t.Fatal("bad trial definitions returned")
			}
		}
	})
	t.Run("List Experiments", func(t *testing.T) {
		experiments, err := db.ListExperiments(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, experiment := range experiments {
			if experiment.ID != ids[0] && experiment.ID != ids[1] {
				t.Fatal("bad experiment id found")
			}
		}
	})
	t.Run("Update Experiments", func(t *testing.T) {
		for _, id := range ids {
			exp, err := db.GetExperiment(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			prevUpdateAt := exp.UpdatedAt
			exp.Labels = append(exp.Labels, "test label")
			exp, err = db.UpdateExperiment(ctx, exp)
			if err != nil {
				t.Fatal(err)
			}
			if exp.UpdatedAt.Before(prevUpdateAt) {
				t.Fatal("bad update at time")
			}
		}
	})
	t.Run("Label Experiments", func(t *testing.T) {
		exps, err := db.LabelExperiments(
			ctx,
			ids,
			[]string{"should be present"},
			[]string{"test label"},
		)
		if err != nil {
			t.Fatal(err)
		}
		for _, exp := range exps {
			if len(exp.Labels) != 1 {
				t.Fatal("bad number of labels")
			}
			if exp.Labels[0] != "should be present" {
				t.Fatal("bad label found")
			}
		}
	})
	t.Run("Delete Experiment", func(t *testing.T) {
		if err := db.DeleteExperiment(ctx, ids[0]); err != nil {
			t.Fatal(err)
		}
		if _, err := db.GetExperiment(ctx, ids[0]); err == nil {
			t.Fatal("error expected")
		}
		if _, err := db.GetExperiment(ctx, ids[1]); err != nil {
			t.Fatal(err)
		}
	})
}

func newTestExperiment(t *testing.T, sourceFile, name string) metadata.Experiment {
	data, err := ioutil.ReadFile("../cue/cue.mod/p2plab.cue")
	if err != nil {
		t.Fatal(err)
	}
	sourceData, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	psr := parser.NewParser([]string{string(data)})
	inst, err := psr.Compile(
		name,
		string(sourceData),
	)
	if err != nil {
		t.Fatal(err)
	}
	edef, err := inst.ToExperimentDefinition()
	if err != nil {
		t.Fatal(err)
	}
	return metadata.Experiment{
		ID:         uuid.New().String(),
		Status:     metadata.ExperimentRunning,
		Definition: edef,
	}
}

func newTestDB(t *testing.T, path string) (metadata.DB, func() error) {
	db, err := metadata.NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() error {
		if err := db.Close(); err != nil {
			return err
		}
		return os.RemoveAll(path)
	}
	return db, cleanup
}
