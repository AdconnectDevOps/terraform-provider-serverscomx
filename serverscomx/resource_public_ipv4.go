package serverscomx

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// publicIPv4MaxPollAttempts bounds the post-allocation poll loop. The rate
// limiter spaces each GetNetwork ~request_interval (default 1s) apart, so this
// is roughly the max wait in seconds. Live allocations finish in ~1s.
const publicIPv4MaxPollAttempts = 60

var (
	_ resource.Resource                = &PublicIPv4Resource{}
	_ resource.ResourceWithImportState = &PublicIPv4Resource{}
)

type PublicIPv4Resource struct {
	client *Client
}

type PublicIPv4ResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	HostID             types.String `tfsdk:"host_id"`
	Mask               types.Int64  `tfsdk:"mask"`
	DistributionMethod types.String `tfsdk:"distribution_method"`
	CIDR               types.String `tfsdk:"cidr"`
	IPAddress          types.String `tfsdk:"ip_address"`
}

func NewPublicIPv4Resource() resource.Resource {
	return &PublicIPv4Resource{}
}

func (r *PublicIPv4Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_ipv4"
}

func (r *PublicIPv4Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Servers.com exposes no PUT/PATCH for network allocation — host_id, mask and
	// distribution_method are all RequiresReplace, so any change is delete+create.
	resp.Schema = schema.Schema{
		Description: "Allocates an additional public IPv4 (alias) address on a Servers.com dedicated server via POST /hosts/dedicated_servers/{host_id}/networks/public_ipv4. Removal deallocates the address.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Servers.com-assigned network ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_id": schema.StringAttribute{
				Description: "Servers.com dedicated server ID (8-character token). Find via GET /hosts/dedicated_servers?search_pattern=<hostname>. Changing this forces a new allocation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mask": schema.Int64Attribute{
				Description: "Prefix length to allocate. Defaults to 32 (a single alias address). Changing this forces a new allocation.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(32),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"distribution_method": schema.StringAttribute{
				Description: "How the address is bound to the server: \"route\" (default, alias IP) or \"gateway\". Changing this forces a new allocation.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("route"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cidr": schema.StringAttribute{
				Description: "Allocated network in CIDR notation, e.g. \"203.0.113.10/32\". Assigned asynchronously after creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_address": schema.StringAttribute{
				Description: "Bare allocated address without the prefix, e.g. \"203.0.113.10\". Convenient for feeding serverscomx_ptr_record.ip.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *PublicIPv4Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected *serverscomx.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *PublicIPv4Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PublicIPv4ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := PublicIPv4CreateRequest{
		DistributionMethod: plan.DistributionMethod.ValueString(),
		Mask:               plan.Mask.ValueInt64(),
	}

	created, err := r.client.CreatePublicIPv4(plan.HostID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Failed to allocate public IPv4", err.Error())
		return
	}

	net, err := r.client.WaitForNetworkActive(plan.HostID.ValueString(), created.ID, publicIPv4MaxPollAttempts)
	if err != nil {
		// The address may have been allocated but not yet active. Surface the id
		// so the operator can inspect/clean up rather than leaking an orphan.
		resp.Diagnostics.AddError(
			"Public IPv4 allocation did not complete",
			fmt.Sprintf("network id %s on host %s: %s", created.ID, plan.HostID.ValueString(), err.Error()),
		)
		return
	}

	r.apply(&plan, net)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PublicIPv4Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PublicIPv4ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	net, err := r.client.GetNetwork(state.HostID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read public IPv4", err.Error())
		return
	}

	// A deallocated address lingers as a tombstone with status "removed" — treat
	// it as gone so the next plan recreates it.
	if net.Status == "removed" || net.Status == "removing" {
		resp.State.RemoveResource(ctx)
		return
	}

	r.apply(&state, net)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op — every user-supplied attribute is RequiresReplace, so the
// framework short-circuits to Delete+Create. Required only to satisfy the
// resource.Resource interface; carry plan forward defensively.
func (r *PublicIPv4Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PublicIPv4ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PublicIPv4Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PublicIPv4ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteNetwork(state.HostID.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to deallocate public IPv4", err.Error())
		return
	}
}

// ImportState accepts the composite key "host_id:network_id" — networks are
// per-host, with no global namespace. cidr/ip_address are filled by Read.
func (r *PublicIPv4Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format \"host_id:network_id\", got %q.", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("host_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// apply copies API-returned network fields into the model. Mask is derived from
// the CIDR prefix — the API has no mask field, and without setting it here an
// imported resource would have a null mask, diffing against the default 32 and
// (because mask is RequiresReplace) proposing a destroy+recreate on first plan.
func (r *PublicIPv4Resource) apply(m *PublicIPv4ResourceModel, net *Network) {
	m.ID = types.StringValue(net.ID)
	m.CIDR = types.StringValue(net.CIDR)
	m.IPAddress = types.StringValue(addressOf(net.CIDR))
	if mask, ok := maskOf(net.CIDR); ok {
		m.Mask = types.Int64Value(mask)
	}
	if net.DistributionMethod != "" {
		m.DistributionMethod = types.StringValue(net.DistributionMethod)
	}
}

// addressOf strips the prefix length from a CIDR ("203.0.113.10/32" -> "203.0.113.10").
func addressOf(cidr string) string {
	if i := strings.IndexByte(cidr, '/'); i >= 0 {
		return cidr[:i]
	}
	return cidr
}

// maskOf extracts the prefix length from a CIDR ("203.0.113.10/32" -> 32).
func maskOf(cidr string) (int64, bool) {
	i := strings.IndexByte(cidr, '/')
	if i < 0 || i+1 >= len(cidr) {
		return 0, false
	}
	n, err := strconv.ParseInt(cidr[i+1:], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
