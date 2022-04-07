package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"go.step.sm/linkedca"

	"github.com/smallstep/assert"
	"github.com/smallstep/certificates/authority/admin"
)

func readProtoJSON(r io.ReadCloser, m proto.Message) error {
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return protojson.Unmarshal(data, m)
}

func TestHandler_requireEABEnabled(t *testing.T) {
	type test struct {
		ctx        context.Context
		next       http.HandlerFunc
		err        *admin.Error
		statusCode int
	}
	var tests = map[string]func(t *testing.T) test{
		"fail/prov.GetDetails": func(t *testing.T) test {
			prov := &linkedca.Provisioner{
				Id:   "provID",
				Name: "provName",
			}
			ctx := linkedca.NewContextWithProvisioner(context.Background(), prov)
			err := admin.NewErrorISE("error getting details for provisioner 'provName'")
			err.Message = "error getting details for provisioner 'provName'"
			return test{
				ctx:        ctx,
				err:        err,
				statusCode: 500,
			}
		},
		"fail/details.GetACME": func(t *testing.T) test {
			prov := &linkedca.Provisioner{
				Id:      "provID",
				Name:    "provName",
				Details: &linkedca.ProvisionerDetails{},
			}
			ctx := linkedca.NewContextWithProvisioner(context.Background(), prov)
			err := admin.NewErrorISE("error getting ACME details for provisioner 'provName'")
			err.Message = "error getting ACME details for provisioner 'provName'"
			return test{
				ctx:        ctx,
				err:        err,
				statusCode: 500,
			}
		},
		"ok/eab-disabled": func(t *testing.T) test {
			prov := &linkedca.Provisioner{
				Id:   "provID",
				Name: "provName",
				Details: &linkedca.ProvisionerDetails{
					Data: &linkedca.ProvisionerDetails_ACME{
						ACME: &linkedca.ACMEProvisioner{
							RequireEab: false,
						},
					},
				},
			}
			ctx := linkedca.NewContextWithProvisioner(context.Background(), prov)
			err := admin.NewError(admin.ErrorBadRequestType, "ACME EAB not enabled for provisioner provName")
			err.Message = "ACME EAB not enabled for provisioner 'provName'"
			return test{
				ctx:        ctx,
				err:        err,
				statusCode: 400,
			}
		},
		"ok/eab-enabled": func(t *testing.T) test {
			prov := &linkedca.Provisioner{
				Id:   "provID",
				Name: "provName",
				Details: &linkedca.ProvisionerDetails{
					Data: &linkedca.ProvisionerDetails_ACME{
						ACME: &linkedca.ACMEProvisioner{
							RequireEab: true,
						},
					},
				},
			}
			ctx := linkedca.NewContextWithProvisioner(context.Background(), prov)
			return test{
				ctx: ctx,
				next: func(w http.ResponseWriter, r *http.Request) {
					w.Write(nil) // mock response with status 200
				},
				statusCode: 200,
			}
		},
	}

	for name, prep := range tests {
		tc := prep(t)
		t.Run(name, func(t *testing.T) {
			h := &Handler{}

			req := httptest.NewRequest("GET", "/foo", nil)
			req = req.WithContext(tc.ctx)
			w := httptest.NewRecorder()
			h.requireEABEnabled(tc.next)(w, req)
			res := w.Result()

			assert.Equals(t, tc.statusCode, res.StatusCode)

			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			assert.FatalError(t, err)

			if res.StatusCode >= 400 {
				err := admin.Error{}
				assert.FatalError(t, json.Unmarshal(bytes.TrimSpace(body), &err))

				assert.Equals(t, tc.err.Type, err.Type)
				assert.Equals(t, tc.err.Message, err.Message)
				assert.Equals(t, tc.err.StatusCode(), res.StatusCode)
				assert.Equals(t, tc.err.Detail, err.Detail)
				assert.Equals(t, []string{"application/json"}, res.Header["Content-Type"])
				return
			}
		})
	}
}

func TestCreateExternalAccountKeyRequest_Validate(t *testing.T) {
	type fields struct {
		Reference string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "fail/reference-too-long",
			fields: fields{
				Reference: strings.Repeat("A", 257),
			},
			wantErr: true,
		},
		{
			name: "ok/empty-reference",
			fields: fields{
				Reference: "",
			},
			wantErr: false,
		},
		{
			name: "ok",
			fields: fields{
				Reference: "my-eab-reference",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CreateExternalAccountKeyRequest{
				Reference: tt.fields.Reference,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("CreateExternalAccountKeyRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_CreateExternalAccountKey(t *testing.T) {
	type test struct {
		ctx        context.Context
		statusCode int
		err        *admin.Error
	}
	var tests = map[string]func(t *testing.T) test{
		"ok": func(t *testing.T) test {
			chiCtx := chi.NewRouteContext()
			ctx := context.WithValue(context.Background(), chi.RouteCtxKey, chiCtx)
			return test{
				ctx:        ctx,
				statusCode: 501,
				err: &admin.Error{
					Type:    admin.ErrorNotImplementedType.String(),
					Status:  http.StatusNotImplemented,
					Message: "this functionality is currently only available in Certificate Manager: https://u.step.sm/cm",
					Detail:  "not implemented",
				},
			}
		},
	}
	for name, prep := range tests {
		tc := prep(t)
		t.Run(name, func(t *testing.T) {

			req := httptest.NewRequest("POST", "/foo", nil) // chi routing is prepared in test setup
			req = req.WithContext(tc.ctx)
			w := httptest.NewRecorder()
			acmeResponder := NewACMEAdminResponder()
			acmeResponder.CreateExternalAccountKey(w, req)
			res := w.Result()
			assert.Equals(t, tc.statusCode, res.StatusCode)

			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			assert.FatalError(t, err)

			adminErr := admin.Error{}
			assert.FatalError(t, json.Unmarshal(bytes.TrimSpace(body), &adminErr))

			assert.Equals(t, tc.err.Type, adminErr.Type)
			assert.Equals(t, tc.err.Message, adminErr.Message)
			assert.Equals(t, tc.err.StatusCode(), res.StatusCode)
			assert.Equals(t, tc.err.Detail, adminErr.Detail)
			assert.Equals(t, []string{"application/json"}, res.Header["Content-Type"])

		})
	}
}

func TestHandler_DeleteExternalAccountKey(t *testing.T) {
	type test struct {
		ctx        context.Context
		statusCode int
		err        *admin.Error
	}
	var tests = map[string]func(t *testing.T) test{
		"ok": func(t *testing.T) test {
			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("provisionerName", "provName")
			chiCtx.URLParams.Add("id", "keyID")
			ctx := context.WithValue(context.Background(), chi.RouteCtxKey, chiCtx)
			return test{
				ctx:        ctx,
				statusCode: 501,
				err: &admin.Error{
					Type:    admin.ErrorNotImplementedType.String(),
					Status:  http.StatusNotImplemented,
					Message: "this functionality is currently only available in Certificate Manager: https://u.step.sm/cm",
					Detail:  "not implemented",
				},
			}
		},
	}
	for name, prep := range tests {
		tc := prep(t)
		t.Run(name, func(t *testing.T) {

			req := httptest.NewRequest("DELETE", "/foo", nil) // chi routing is prepared in test setup
			req = req.WithContext(tc.ctx)
			w := httptest.NewRecorder()
			acmeResponder := NewACMEAdminResponder()
			acmeResponder.DeleteExternalAccountKey(w, req)
			res := w.Result()
			assert.Equals(t, tc.statusCode, res.StatusCode)

			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			assert.FatalError(t, err)

			adminErr := admin.Error{}
			assert.FatalError(t, json.Unmarshal(bytes.TrimSpace(body), &adminErr))

			assert.Equals(t, tc.err.Type, adminErr.Type)
			assert.Equals(t, tc.err.Message, adminErr.Message)
			assert.Equals(t, tc.err.StatusCode(), res.StatusCode)
			assert.Equals(t, tc.err.Detail, adminErr.Detail)
			assert.Equals(t, []string{"application/json"}, res.Header["Content-Type"])
		})
	}
}

func TestHandler_GetExternalAccountKeys(t *testing.T) {
	type test struct {
		ctx        context.Context
		statusCode int
		req        *http.Request
		err        *admin.Error
	}
	var tests = map[string]func(t *testing.T) test{
		"ok": func(t *testing.T) test {
			chiCtx := chi.NewRouteContext()
			chiCtx.URLParams.Add("provisionerName", "provName")
			req := httptest.NewRequest("GET", "/foo", nil)
			ctx := context.WithValue(context.Background(), chi.RouteCtxKey, chiCtx)
			return test{
				ctx:        ctx,
				statusCode: 501,
				req:        req,
				err: &admin.Error{
					Type:    admin.ErrorNotImplementedType.String(),
					Status:  http.StatusNotImplemented,
					Message: "this functionality is currently only available in Certificate Manager: https://u.step.sm/cm",
					Detail:  "not implemented",
				},
			}
		},
	}
	for name, prep := range tests {
		tc := prep(t)
		t.Run(name, func(t *testing.T) {

			req := tc.req.WithContext(tc.ctx)
			w := httptest.NewRecorder()
			acmeResponder := NewACMEAdminResponder()
			acmeResponder.GetExternalAccountKeys(w, req)

			res := w.Result()
			assert.Equals(t, tc.statusCode, res.StatusCode)

			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			assert.FatalError(t, err)

			adminErr := admin.Error{}
			assert.FatalError(t, json.Unmarshal(bytes.TrimSpace(body), &adminErr))

			assert.Equals(t, tc.err.Type, adminErr.Type)
			assert.Equals(t, tc.err.Message, adminErr.Message)
			assert.Equals(t, tc.err.StatusCode(), res.StatusCode)
			assert.Equals(t, tc.err.Detail, adminErr.Detail)
			assert.Equals(t, []string{"application/json"}, res.Header["Content-Type"])
		})
	}
}
