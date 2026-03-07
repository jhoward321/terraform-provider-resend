package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jhoward321/terraform-provider-resend/internal/client"
)

var _ resource.Resource = &DomainResource{}

type DomainResource struct {
	client *client.Client
}

type DomainResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Region    types.String `tfsdk:"region"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	Records   types.List   `tfsdk:"records"`
}

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Resend sending domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The domain name (e.g., `example.com`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "AWS region for the domain (e.g., `us-east-1`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Domain verification status.",
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"records": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "DNS records required for domain verification.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type":     schema.StringAttribute{Computed: true},
						"name":     schema.StringAttribute{Computed: true},
						"value":    schema.StringAttribute{Computed: true},
						"priority": schema.StringAttribute{Computed: true},
						"ttl":      schema.StringAttribute{Computed: true},
						"status":   schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.Client)
}

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := client.CreateDomainRequest{
		Name:   data.Name.ValueString(),
		Region: data.Region.ValueString(),
	}

	result, err := r.client.CreateDomain(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	data.ID = types.StringValue(result.ID)
	data.Name = types.StringValue(result.Name)
	data.Status = types.StringValue(result.Status)
	data.Region = types.StringValue(result.Region)
	data.CreatedAt = types.StringValue(result.CreatedAt)

	records, diags := dnsRecordsToList(result.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Records = records

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetDomain(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	data.Name = types.StringValue(result.Name)
	data.Status = types.StringValue(result.Status)
	data.Region = types.StringValue(result.Region)
	data.CreatedAt = types.StringValue(result.CreatedAt)

	records, diags := dnsRecordsToList(result.Records)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Records = records

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes require replacement; update is never called.
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDomain(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting domain", err.Error())
		return
	}
}

var dnsRecordAttrTypes = map[string]attr.Type{
	"type":     types.StringType,
	"name":     types.StringType,
	"value":    types.StringType,
	"priority": types.StringType,
	"ttl":      types.StringType,
	"status":   types.StringType,
}

func dnsRecordsToList(records []client.DNSRecord) (types.List, diag.Diagnostics) {
	recordType := types.ObjectType{AttrTypes: dnsRecordAttrTypes}

	if len(records) == 0 {
		return types.ListValueMust(recordType, []attr.Value{}), nil
	}

	var recordObjects []attr.Value
	for _, rec := range records {
		obj, diags := types.ObjectValue(dnsRecordAttrTypes, map[string]attr.Value{
			"type":     types.StringValue(rec.Type),
			"name":     types.StringValue(rec.Name),
			"value":    types.StringValue(rec.Value),
			"priority": types.StringValue(rec.Priority.String()),
			"ttl":      types.StringValue(rec.TTL),
			"status":   types.StringValue(rec.Status),
		})
		if diags.HasError() {
			return types.ListNull(recordType), diags
		}
		recordObjects = append(recordObjects, obj)
	}

	return types.ListValue(recordType, recordObjects)
}
