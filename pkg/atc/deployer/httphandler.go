package deployer

import (
	"fmt"
	"net/http"
)

func (f *Deployer) OkHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	}
}
