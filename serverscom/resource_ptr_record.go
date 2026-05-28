package serverscom

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &PtrRecordResource{}
	_ resource.ResourceWithImportState = &PtrRecordResource{}
)

type PtrRecordResource struct {
	client *Client
}

type PtrRecordResourceModel struct {
	ID       types.String `tfsdk:"id"`
	HostID   types.String `tfsdk:"host_id"`
	IP       types.String `tfsdk:"ip"`
	Domain   types.String `tfsdk:"domain"`
	Priority types.Int64  `tfsdk:"priority"`
	TTL      types.Int64  `tfsdk:"ttl"`
}

func NewPtrRecordResource() resource.Resource {
	return &PtrRecordResource{}
}

func (r *PtrRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ptr_record"
}

func (r *PtrRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// All user-supplied attrs are RequiresReplace — the Servers.com API allows
	// only OPTIONS+DELETE on /ptr_records/{id}, so any change is a delete+create.
	resp.Schema = schema.Schema{
		Description: "Manages a PTR (reverse DNS) record on a Servers.com dedicated server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Servers.com-assigned PTR record ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_id": schema.StringAttribute{
				Description: "Servers.com dedicated server ID (8-character token). Find via GET /hosts/dedicated_servers?search_pattern=<hostname>.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip": schema.StringAttribute{
				Description: "Public IPv4 address to attach the PTR to. Must already be allocated to host_id (alias or primary).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "Fully-qualified domain name returned by reverse DNS for ip.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"priority": schema.Int64Attribute{
				Description: "PTR priority. Defaults to 0 if omitted. Changes force replacement.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"ttl": schema.Int64Attribute{
				Description: "PTR TTL in seconds. Defaults to 60 if omitted. Changes force replacement.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *PtrRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *serverscom.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *PtrRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PtrRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := PtrCreateRequest{
		IP:     plan.IP.ValueString(),
		Domain: plan.Domain.ValueString(),
	}
	if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
		v := plan.Priority.ValueInt64()
		payload.Priority = &v
	}
	if !plan.TTL.IsNull() && !plan.TTL.IsUnknown() {
		v := plan.TTL.ValueInt64()
		payload.TTL = &v
	}

	created, err := r.client.CreatePtrRecord(plan.HostID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create PTR record", err.Error())
		return
	}

	plan.ID = types.StringValue(created.ID)
	plan.Priority = types.Int64Value(created.Priority)
	plan.TTL = types.Int64Value(created.TTL)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PtrRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PtrRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rec, err := r.client.GetPtrRecord(state.HostID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read PTR record", err.Error())
		return
	}

	state.IP = types.StringValue(rec.IP)
	state.Domain = types.StringValue(rec.Domain)
	state.Priority = types.Int64Value(rec.Priority)
	state.TTL = types.Int64Value(rec.TTL)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op — every schema attribute is RequiresReplace. The framework
// short-circuits to Delete+Create before reaching this method. Implementing it
// is required only to satisfy the resource.Resource interface.
func (r *PtrRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PtrRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PtrRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PtrRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeletePtrRecord(state.HostID.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete PTR record", err.Error())
		return
	}
}

// ImportState accepts a composite key "host_id:ptr_id" since the Servers.com
// API has no global PTR namespace — records are per-host.
func (r *PtrRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format \"host_id:ptr_id\", got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("host_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
