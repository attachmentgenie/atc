package atc

import (
	"net/http"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
)

func (t *Atc) servicesHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "text/plain")

	svcNames := make([]string, 0, len(t.ServiceMap))
	for name := range t.ServiceMap {
		svcNames = append(svcNames, name)
	}

	sort.Strings(svcNames)

	x := table.NewWriter()
	x.SetOutputMirror(w)
	x.AppendHeader(table.Row{"service name", "status", "failure case"})

	for _, name := range svcNames {
		service := t.ServiceMap[name]

		var e string

		if err := service.FailureCase(); err != nil {
			e = err.Error()
		}

		x.AppendRows([]table.Row{
			{name, service.State(), e},
		})
	}

	x.AppendSeparator()
	x.Render()
}
