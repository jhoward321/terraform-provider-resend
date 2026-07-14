package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jhoward321/terraform-provider-resend/internal/client"
)

var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

type DomainResource struct {
	client *client.Client
}

type DomainResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Region            types.String `tfsdk:"region"`
	Status            types.String `tfsdk:"status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	CustomReturnPath  types.String `tfsdk:"custom_return_path"`
	OpenTracking      types.Bool   `tfsdk:"open_tracking"`
	ClickTracking     types.Bool   `tfsdk:"click_tracking"`
	TrackingSubdomain types.String `tfsdk:"tracking_subdomain"`
	TLS               types.String `tfsdk:"tls"`
	Capabilities      types.Object `tfsdk:"capabilities"`
	SPFMXRecord       types.Object `tfsdk:"spf_mx_record"`
	SPFTXTRecord      types.Object `tfsdk:"spf_txt_record"`
	DKIMRecords       types.List   `tfsdk:"dkim_records"`
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
			"custom_return_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Subdomain for the custom Return-Path (SPF/DMARC). Defaults to `send`. Create-only. Write-only in the Resend API (not returned on read), so drift is not detected.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"open_tracking": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Track email opens. Updated in place.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"click_tracking": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Track clicks within HTML email bodies. Updated in place.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"tracking_subdomain": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Custom subdomain used for open/click tracking. Updated in place.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tls": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "TLS enforcement policy. One of `opportunistic` or `enforced`. Updated in place. Write-only in the Resend API (not returned on read), so drift is not detected.",
				Validators: []validator.String{
					stringvalidator.OneOf("opportunistic", "enforced"),
				},
			},
			"capabilities": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Domain sending/receiving capabilities. At least one must be `enabled` (enforced by the API).",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"sending": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "`enabled` or `disabled`. Defaults to `enabled`.",
						Validators: []validator.String{
							stringvalidator.OneOf("enabled", "disabled"),
						},
					},
					"receiving": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "`enabled` or `disabled`. Defaults to `disabled`.",
						Validators: []validator.String{
							stringvalidator.OneOf("enabled", "disabled"),
						},
					},
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
	if !data.CustomReturnPath.IsNull() {
		v := data.CustomReturnPath.ValueString()
		apiReq.CustomReturnPath = &v
	}
	if !data.OpenTracking.IsNull() && !data.OpenTracking.IsUnknown() {
		v := data.OpenTracking.ValueBool()
		apiReq.OpenTracking = &v
	}
	if !data.ClickTracking.IsNull() && !data.ClickTracking.IsUnknown() {
		v := data.ClickTracking.ValueBool()
		apiReq.ClickTracking = &v
	}
	if !data.TrackingSubdomain.IsNull() && !data.TrackingSubdomain.IsUnknown() {
		v := data.TrackingSubdomain.ValueString()
		apiReq.TrackingSubdomain = &v
	}
	if !data.TLS.IsNull() {
		v := data.TLS.ValueString()
		apiReq.TLS = &v
	}
	apiReq.Capabilities = capabilitiesToAPI(data.Capabilities)

	result, err := r.client.CreateDomain(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	resp.Diagnostics.Append(setDomainState(result, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// TLS and CustomReturnPath are preserved from the plan (write-only in the API).
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

	// TLS and CustomReturnPath are left as-is in state (write-only in the API).
	resp.Diagnostics.Append(setDomainState(result, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updReq := client.UpdateDomainRequest{}
	if !data.OpenTracking.IsNull() && !data.OpenTracking.IsUnknown() {
		v := data.OpenTracking.ValueBool()
		updReq.OpenTracking = &v
	}
	if !data.ClickTracking.IsNull() && !data.ClickTracking.IsUnknown() {
		v := data.ClickTracking.ValueBool()
		updReq.ClickTracking = &v
	}
	if !data.TrackingSubdomain.IsNull() && !data.TrackingSubdomain.IsUnknown() {
		v := data.TrackingSubdomain.ValueString()
		updReq.TrackingSubdomain = &v
	}
	if !data.TLS.IsNull() {
		v := data.TLS.ValueString()
		updReq.TLS = &v
	}
	updReq.Capabilities = capabilitiesToAPI(data.Capabilities)

	result, err := r.client.UpdateDomain(ctx, data.ID.ValueString(), updReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	resp.Diagnostics.Append(setDomainState(result, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// TLS and CustomReturnPath are preserved from the plan (write-only in the API).
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
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

var capabilitiesAttrTypes = map[string]attr.Type{
	"sending":   types.StringType,
	"receiving": types.StringType,
}

// capabilitiesToAPI builds a *client.Capabilities from the nested object, or
// nil when the object is null/unknown (so it is omitted from the request).
func capabilitiesToAPI(obj types.Object) *client.Capabilities {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	attrs := obj.Attributes()
	caps := &client.Capabilities{}
	if s, ok := attrs["sending"].(types.String); ok && !s.IsNull() && !s.IsUnknown() {
		caps.Sending = s.ValueString()
	}
	if s, ok := attrs["receiving"].(types.String); ok && !s.IsNull() && !s.IsUnknown() {
		caps.Receiving = s.ValueString()
	}
	return caps
}

func capabilitiesToObject(c client.Capabilities) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(capabilitiesAttrTypes, map[string]attr.Value{
		"sending":   types.StringValue(c.Sending),
		"receiving": types.StringValue(c.Receiving),
	})
}

// setDomainState copies all API-readable fields from result into data. It does
// NOT touch TLS or CustomReturnPath, which the API never returns (write-only).
func setDomainState(result *client.Domain, data *DomainResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	data.ID = types.StringValue(result.ID)
	data.Name = types.StringValue(result.Name)
	data.Status = types.StringValue(result.Status)
	data.Region = types.StringValue(result.Region)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.OpenTracking = types.BoolValue(result.OpenTracking)
	data.ClickTracking = types.BoolValue(result.ClickTracking)
	data.TrackingSubdomain = types.StringValue(result.TrackingSubdomain)

	capsObj, d := capabilitiesToObject(result.Capabilities)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	data.Capabilities = capsObj

	diags.Append(setDNSRecordState(result.Records, data)...)
	return diags
}
