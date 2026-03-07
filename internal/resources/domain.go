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
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Region       types.String `tfsdk:"region"`
	Status       types.String `tfsdk:"status"`
	CreatedAt    types.String `tfsdk:"created_at"`
	SPFMXRecord  types.Object `tfsdk:"spf_mx_record"`
	SPFTXTRecord types.Object `tfsdk:"spf_txt_record"`
	DKIMRecords  types.List   `tfsdk:"dkim_records"`
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
				Computed:            true,
				MarkdownDescription: "Domain identifier.",
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
				Computed:            true,
				MarkdownDescription: "Timestamp when the domain was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"spf_mx_record": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "SPF MX record for domain verification.",
				Attributes: map[string]schema.Attribute{
					"record":   schema.StringAttribute{Computed: true, MarkdownDescription: "Record purpose (e.g., `SPF`, `DKIM`)."},
					"type":     schema.StringAttribute{Computed: true, MarkdownDescription: "DNS record type (e.g., `MX`, `TXT`, `CNAME`)."},
					"name":     schema.StringAttribute{Computed: true, MarkdownDescription: "DNS record hostname."},
					"value":    schema.StringAttribute{Computed: true, MarkdownDescription: "DNS record value."},
					"priority": schema.StringAttribute{Computed: true, MarkdownDescription: "MX record priority."},
					"ttl":      schema.StringAttribute{Computed: true, MarkdownDescription: "Time to live."},
					"status":   schema.StringAttribute{Computed: true, MarkdownDescription: "Verification status of this record."},
				},
			},
			"spf_txt_record": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "SPF TXT record for domain verification.",
				Attributes: map[string]schema.Attribute{
					"record": schema.StringAttribute{Computed: true, MarkdownDescription: "Record purpose (e.g., `SPF`, `DKIM`)."},
					"type":   schema.StringAttribute{Computed: true, MarkdownDescription: "DNS record type (e.g., `MX`, `TXT`, `CNAME`)."},
					"name":   schema.StringAttribute{Computed: true, MarkdownDescription: "DNS record hostname."},
					"value":  schema.StringAttribute{Computed: true, MarkdownDescription: "DNS record value."},
					"ttl":    schema.StringAttribute{Computed: true, MarkdownDescription: "Time to live."},
					"status": schema.StringAttribute{Computed: true, MarkdownDescription: "Verification status of this record."},
				},
			},
			"dkim_records": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "DKIM records for domain verification.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"record": schema.StringAttribute{Computed: true, MarkdownDescription: "Record purpose (e.g., `SPF`, `DKIM`)."},
						"type":   schema.StringAttribute{Computed: true, MarkdownDescription: "DNS record type (e.g., `MX`, `TXT`, `CNAME`)."},
						"name":   schema.StringAttribute{Computed: true, MarkdownDescription: "DNS record hostname."},
						"value":  schema.StringAttribute{Computed: true, MarkdownDescription: "DNS record value."},
						"ttl":    schema.StringAttribute{Computed: true, MarkdownDescription: "Time to live."},
						"status": schema.StringAttribute{Computed: true, MarkdownDescription: "Verification status of this record."},
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

	resp.Diagnostics.Append(setDNSRecordState(result.Records, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	resp.Diagnostics.Append(setDNSRecordState(result.Records, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

var spfMXRecordAttrTypes = map[string]attr.Type{
	"record":   types.StringType,
	"type":     types.StringType,
	"name":     types.StringType,
	"value":    types.StringType,
	"priority": types.StringType,
	"ttl":      types.StringType,
	"status":   types.StringType,
}

var dkimRecordAttrTypes = map[string]attr.Type{
	"record": types.StringType,
	"type":   types.StringType,
	"name":   types.StringType,
	"value":  types.StringType,
	"ttl":    types.StringType,
	"status": types.StringType,
}

var spfTXTRecordAttrTypes = dkimRecordAttrTypes

func splitDNSRecords(records []client.DNSRecord) (spfMX *client.DNSRecord, spfTXT *client.DNSRecord, dkim []client.DNSRecord) {
	for i := range records {
		rec := &records[i]
		switch {
		case rec.Record == "SPF" && rec.Type == "MX":
			spfMX = rec
		case rec.Record == "SPF" && rec.Type == "TXT":
			spfTXT = rec
		case rec.Record == "DKIM":
			dkim = append(dkim, *rec)
		}
	}
	return
}

func spfMXToObject(rec *client.DNSRecord) (types.Object, diag.Diagnostics) {
	if rec == nil {
		return types.ObjectNull(spfMXRecordAttrTypes), nil
	}
	return types.ObjectValue(spfMXRecordAttrTypes, map[string]attr.Value{
		"record":   types.StringValue(rec.Record),
		"type":     types.StringValue(rec.Type),
		"name":     types.StringValue(rec.Name),
		"value":    types.StringValue(rec.Value),
		"priority": types.StringValue(rec.Priority.String()),
		"ttl":      types.StringValue(rec.TTL),
		"status":   types.StringValue(rec.Status),
	})
}

func dnsRecordToObject(rec client.DNSRecord) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(dkimRecordAttrTypes, map[string]attr.Value{
		"record": types.StringValue(rec.Record),
		"type":   types.StringValue(rec.Type),
		"name":   types.StringValue(rec.Name),
		"value":  types.StringValue(rec.Value),
		"ttl":    types.StringValue(rec.TTL),
		"status": types.StringValue(rec.Status),
	})
}

func dkimRecordsToList(records []client.DNSRecord) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: dkimRecordAttrTypes}
	if len(records) == 0 {
		return types.ListValueMust(objType, []attr.Value{}), nil
	}
	var objs []attr.Value
	for _, rec := range records {
		obj, diags := dnsRecordToObject(rec)
		if diags.HasError() {
			return types.ListNull(objType), diags
		}
		objs = append(objs, obj)
	}
	return types.ListValue(objType, objs)
}

func setDNSRecordState(records []client.DNSRecord, data *DomainResourceModel) diag.Diagnostics {
	var allDiags diag.Diagnostics

	spfMX, spfTXT, dkim := splitDNSRecords(records)

	spfMXObj, diags := spfMXToObject(spfMX)
	allDiags.Append(diags...)

	var spfTXTObj types.Object
	if spfTXT != nil {
		spfTXTObj, diags = dnsRecordToObject(*spfTXT)
	} else {
		spfTXTObj = types.ObjectNull(spfTXTRecordAttrTypes)
	}
	allDiags.Append(diags...)

	dkimList, diags := dkimRecordsToList(dkim)
	allDiags.Append(diags...)

	if allDiags.HasError() {
		return allDiags
	}

	data.SPFMXRecord = spfMXObj
	data.SPFTXTRecord = spfTXTObj
	data.DKIMRecords = dkimList
	return nil
}
