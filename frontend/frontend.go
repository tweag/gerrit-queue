package frontend

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"html/template"

	"github.com/gin-gonic/gin"
	"github.com/rakyll/statik/fs"

	"github.com/tweag/gerrit-queue/gerrit"
	_ "github.com/tweag/gerrit-queue/statik" // register static assets
	"github.com/tweag/gerrit-queue/submitqueue"
)

// Frontend holds a gin Engine and the Sergequeue object
type Frontend struct {
	Router      *gin.Engine
	SubmitQueue *submitqueue.SubmitQueue
}

//loadTemplate loads a single template from statikFS and returns a template object
func loadTemplate(templateName string, funcMap template.FuncMap) (*template.Template, error) {
	statikFS, err := fs.New()
	if err != nil {
		return nil, err
	}

	tmpl := template.New(templateName).Funcs(funcMap)
	r, err := statikFS.Open("/" + templateName)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return tmpl.Parse(string(contents))
}

// MakeFrontend configures the router and returns a new Frontend struct
func MakeFrontend(runner *submitqueue.Runner) *Frontend {
	router := gin.Default()

	submitQueue := runner.GetSubmitQueue()

	funcMap := template.FuncMap{
		"isAutoSubmittable": func(serie *submitqueue.Serie) bool {
			return submitQueue.IsAutoSubmittable(serie)
		},
		"changesetURL": func(changeset *gerrit.Changeset) string {
			return submitQueue.GetChangesetURL(changeset)
		},
	}

	tmpl := template.Must(loadTemplate("submit-queue.tmpl.html", funcMap))

	router.SetHTMLTemplate(tmpl)

	router.GET("/submit-queue.json", func(c *gin.Context) {

		// FIXME: do this periodically
		err := submitQueue.UpdateHEAD()
		if err != nil {
			c.AbortWithError(http.StatusBadGateway, fmt.Errorf("unable to update HEAD"))
		}
		c.JSON(http.StatusOK, submitQueue)
	})

	router.GET("/", func(c *gin.Context) {
		// FIXME: do this periodically
		// TODO: add hyperlinks to changesets
		err := submitQueue.UpdateHEAD()
		if err != nil {
			c.AbortWithError(http.StatusBadGateway, fmt.Errorf("unable to update HEAD"))
		}

		c.HTML(http.StatusOK, "submit-queue.tmpl.html", gin.H{
			"series":      submitQueue.Series,
			"projectName": submitQueue.ProjectName,
			"branchName":  submitQueue.BranchName,
			"HEAD":        submitQueue.HEAD,
		})
	})
	return &Frontend{
		Router:      router,
		SubmitQueue: submitQueue,
	}
}

func (f *Frontend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.Router.ServeHTTP(w, r)
}
