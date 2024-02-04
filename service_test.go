package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleProcess(t *testing.T) {
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleProcess(tt.args.w, tt.args.r)
		})
	}
}
