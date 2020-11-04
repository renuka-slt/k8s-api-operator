package status

import (
	"github.com/wso2/k8s-api-operator/api-operator/pkg/envoy/controller"
	"reflect"
	"testing"
)

func TestUpdatedProjects(t *testing.T) {
	var tests = []struct {
		name      string
		st, newSt *ProjectsStatus
		want      map[string]bool
	}{
		// Same ingresses
		{
			name:  "No hosts added or deleted",
			st:    &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_", "b_com": "_"}},
			newSt: &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_", "b_com": "_"}},
			want:  map[string]bool{"a_com": true, "b_com": true},
		},
		{
			name:  "Add and delete hosts",
			st:    &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_", "b_com": "_"}},
			newSt: &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_", "c_com": "_"}},
			want:  map[string]bool{"a_com": true, "b_com": true, "c_com": true},
		},
		// Different ingresses
		{
			name:  "Add new ingress",
			st:    &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_", "b_com": "_"}},
			newSt: &ProjectsStatus{"foo/ing2": map[string]string{"b_com": "_", "c_com": "_"}},
			want:  map[string]bool{"b_com": true, "c_com": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.st.UpdatedProjects(tt.newSt)
			if !reflect.DeepEqual(p, tt.want) {
				t.Errorf("%v.UpdatedProjects(%v) = %v; want %v", tt.st, tt.newSt, p, tt.want)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	var tests = []struct {
		name       string
		st, newSt  *ProjectsStatus
		gwResponse controller.Response
		want       *ProjectsStatus
	}{
		{
			name:       "Successful deletion & update",
			st:         &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_", "b_com": "_"}},
			newSt:      &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_"}},
			gwResponse: controller.Response{"a_com": controller.Updated, "b_com": controller.Deleted},
			want:       &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_"}},
		},
		{
			name:       "Failed deletion & update",
			st:         &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_", "b_com": "_"}},
			newSt:      &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_"}},
			gwResponse: controller.Response{"a_com": controller.Failed, "b_com": controller.Failed},
			want:       &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_", "b_com": "_"}},
		},
		{
			name:       "Failed update",
			st:         &ProjectsStatus{"foo/ing1": map[string]string{"a_com": "_"}},
			newSt:      &ProjectsStatus{"foo/ing1": map[string]string{"b_com": "_"}},
			gwResponse: controller.Response{"a_com": controller.Deleted, "b_com": controller.Failed},
			want:       &ProjectsStatus{},
		},
		{
			name: "Mixed operations",
			st: &ProjectsStatus{
				"foo/ing1": map[string]string{"a_com": "_", "b_com": "_"},
				"foo/ing2": map[string]string{"a_com": "_", "c_com": "_", "d_com": "_", "e_com": "_"},
				"foo/ing3": map[string]string{"e_com": "_"},
			},
			newSt: &ProjectsStatus{
				"foo/ing1": map[string]string{"a_com": "_", "c_com": "_"},
				"foo/ing2": map[string]string{"a_com": "_", "c_com": "_", "f_com": "_"},
				"foo/ing4": map[string]string{"f_com": "_"},
				"foo/ing5": map[string]string{"a_com": "_"},
			},
			gwResponse: controller.Response{
				"a_com": controller.Updated,
				"b_com": controller.Deleted,
				"c_com": controller.Failed,
				"d_com": controller.Failed,
				"e_com": controller.Deleted,
				"f_com": controller.Failed,
			},
			want: &ProjectsStatus{
				"foo/ing1": map[string]string{"a_com": "_"},
				"foo/ing2": map[string]string{"a_com": "_", "c_com": "_", "d_com": "_"},
				"foo/ing5": map[string]string{"a_com": "_"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.st.Update(tt.newSt, tt.gwResponse)
			if !reflect.DeepEqual(tt.st, tt.want) {
				t.Errorf("updated state: %v; want %v", tt.st, tt.want)
			}
		})
	}
}
