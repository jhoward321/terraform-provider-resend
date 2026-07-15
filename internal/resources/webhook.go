package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jhoward321/terraform-provider-resend/internal/client"
)

var (
	_ resource.Resource                = &WebhookResource{}
	_ resource.ResourceWithImportState = &WebhookResource{}
)

type WebhookResource struct {
	client *client.Client
}

type WebhookResourceModel struct {
	ID            types.String `tfsdk:"id"`
	URL           types.String `tfsdk:"url"`
	EventTypes    types.List   `tfsdk:"event_types"`
	CreatedAt     types.String `tfsdk:"created_at"`
	SigningSecret types.String `tfsdk:"signing_secret"`
	Status        types.String `tfsdk:"status"`
}

func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

func (r *WebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Resend webhook endpoint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Webhook identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The URL to receive webhook events.",
			},
			"event_types": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Event types to subscribe to (e.g., `email.sent`, `email.delivered`, `email.bounced`).",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the webhook was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"signing_secret": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The webhook signing secret. Only available at creation time; will be empty after `terraform import`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Webhook status. One of `enabled` or `disabled`. Defaults to `enabled`.",
				Default:             stringdefault.StaticString("enabled"),
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
		},
	}
}

func (r *WebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*client.Client)
}

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var eventTypes []string
	resp.Diagnostics.Append(data.EventTypes.ElementsAs(ctx, &eventTypes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateWebhook(ctx, client.CreateWebhookRequest{
		Endpoint: data.URL.ValueString(),
		Events:   eventTypes,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", err.Error())
		return
	}

	data.ID = types.StringValue(result.ID)
	data.SigningSecret = types.StringValue(result.SigningSecret)

	// POST does not accept status; if a non-default status was requested, PATCH it.
	if data.Status.ValueString() == "disabled" {
		if _, err := r.client.UpdateWebhook(ctx, result.ID, client.UpdateWebhookRequest{
			Status: "disabled",
		}); err != nil {
			resp.Diagnostics.AddError("Error setting webhook status", r.rollbackWebhook(ctx, result.ID, err))
			return
		}
	}

	// Create response only returns id and signing_secret, so read back full state.
	webhook, err := r.client.GetWebhook(ctx, result.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook after create", r.rollbackWebhook(ctx, result.ID, err))
		return
	}
	data.URL = types.StringValue(webhook.Endpoint)
	data.CreatedAt = types.StringValue(webhook.CreatedAt)
	data.Status = types.StringValue(webhook.Status)

	etList, diags := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	resp.Diagnostics.Append(diags...)
	data.EventTypes = etList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// rollbackWebhook deletes a webhook that was created during a Create that then
// failed a later step, so it is not left orphaned. It returns an error detail
// describing the original failure plus any cleanup failure.
func (r *WebhookResource) rollbackWebhook(ctx context.Context, id string, cause error) string {
	detail := cause.Error()
	if delErr := r.client.DeleteWebhook(ctx, id); delErr != nil {
		detail = fmt.Sprintf(
			"%s\n\nRolling back the created webhook also failed: %s\nWebhook %q may need to be deleted manually.",
			detail, delErr, id)
	}
	return detail
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetWebhook(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook", err.Error())
		return
	}

	data.URL = types.StringValue(result.Endpoint)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.Status = types.StringValue(result.Status)

	etList, diags := types.ListValueFrom(ctx, types.StringType, result.Events)
	resp.Diagnostics.Append(diags...)
	data.EventTypes = etList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var eventTypes []string
	resp.Diagnostics.Append(data.EventTypes.ElementsAs(ctx, &eventTypes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateWebhook(ctx, data.ID.ValueString(), client.UpdateWebhookRequest{
		Endpoint: data.URL.ValueString(),
		Events:   eventTypes,
		Status:   data.Status.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook", err.Error())
		return
	}

	// Read back to get current state from the API.
	webhook, err := r.client.GetWebhook(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook after update", err.Error())
		return
	}

	data.URL = types.StringValue(webhook.Endpoint)
	data.CreatedAt = types.StringValue(webhook.CreatedAt)
	data.Status = types.StringValue(webhook.Status)

	etList, diags := types.ListValueFrom(ctx, types.StringType, webhook.Events)
	resp.Diagnostics.Append(diags...)
	data.EventTypes = etList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWebhook(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting webhook", err.Error())
		return
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
