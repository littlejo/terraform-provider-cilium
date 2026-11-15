// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestCiliumInstallResourceUpgradeStateV0(t *testing.T) {
	ctx := context.Background()
	r := &CiliumInstallResource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	upgrader := r.UpgradeState(ctx)[0]
	priorType, ok := upgrader.PriorSchema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("unexpected prior schema type: %T", upgrader.PriorSchema.Type().TerraformType(ctx))
	}

	for _, tc := range []struct {
		name     string
		prior    tftypes.Value
		expected bool
	}{
		{name: "true", prior: tftypes.NewValue(tftypes.Bool, true), expected: true},
		{name: "false", prior: tftypes.NewValue(tftypes.Bool, false), expected: false},
		{name: "null", prior: tftypes.NewValue(tftypes.Bool, nil), expected: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attrs := map[string]tftypes.Value{}
			for name, typ := range priorType.AttributeTypes {
				attrs[name] = tftypes.NewValue(typ, nil)
			}
			attrs["id"] = tftypes.NewValue(tftypes.String, "cilium")
			attrs["version"] = tftypes.NewValue(tftypes.String, "1.17.3")
			attrs["reusethenreuse"] = tc.prior

			req := resource.UpgradeStateRequest{
				State: &tfsdk.State{
					Schema: *upgrader.PriorSchema,
					Raw:    tftypes.NewValue(priorType, attrs),
				},
			}
			resp := resource.UpgradeStateResponse{
				State: tfsdk.State{
					Schema: schemaResp.Schema,
					Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
				},
			}
			upgrader.StateUpgrader(ctx, req, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}

			var got CiliumInstallResourceModel
			if diags := resp.State.Get(ctx, &got); diags.HasError() {
				t.Fatalf("unable to read upgraded state: %v", diags)
			}
			if got.ResetThenReuse.ValueBool() != tc.expected {
				t.Errorf("reset_then_reuse = %v, want %v", got.ResetThenReuse, tc.expected)
			}
			if !got.ResetThenReuseLegacy.IsNull() {
				t.Errorf("reusethenreuse = %v, want null", got.ResetThenReuseLegacy)
			}
			if got.Version.ValueString() != "1.17.3" || got.Id.ValueString() != "cilium" {
				t.Errorf("unexpected version/id after upgrade: %v/%v", got.Version, got.Id)
			}
		})
	}
}
