package command

import (
	"errors"
	"github.com/opsgenie/opsgenie-go-sdk-v2/alert"
	gcli "github.com/urfave/cli"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"fmt"
)

func NewAlertClient(c *gcli.Context) (*alert.Client, error) {
	alertCli, cliErr := alert.NewClient(getConfigurations(c))
	if cliErr != nil {
		message := "Can not create the alert client. " + cliErr.Error()
		printMessage(ERROR,message)
		return nil, errors.New(message)
	}
	printMessage(DEBUG,"Alert Client created.")
	return alertCli, nil
}

// CreateAlertAction creates an alert at Opsgenie.
func CreateAlertAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}
	req := alert.CreateAlertRequest{}

	if val, success := getVal("message", c); success {
		req.Message = val
	}
	responders := generateResponders(c, alert.TeamResponder, "teams")
	responders = append(responders, generateResponders(c, alert.UserResponder, "users")...)
	responders = append(responders, generateResponders(c, alert.EscalationResponder, "escalations")...)
	responders = append(responders, generateResponders(c, alert.ScheduleResponder, "schedules")...)

	req.Responders = responders

	if val, success := getVal("alias", c); success {
		req.Alias = val
	}
	if val, success := getVal("actions", c); success {
		req.Actions = strings.Split(val, ",")
	}
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("tags", c); success {
		req.Tags = strings.Split(val, ",")
	}
	if val, success := getVal("description", c); success {
		req.Description = val
	}
	if val, success := getVal("entity", c); success {
		req.Entity = val
	}
	if val, success := getVal("priority", c); success {
		req.Priority = alert.Priority(val)
	}

	req.User = grabUsername(c)

	if val, success := getVal("note", c); success {
		req.Note = val
	}
	if c.IsSet("D") {
		req.Details = extractDetailsFromCommand(c)
	}

	printMessage(DEBUG,"Create alert request prepared from flags, sending request to Opsgenie...")

	resp, err := cli.Create(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Alert will be created.")
	printMessage(INFO, resp.RequestId)
}

func generateResponders(c *gcli.Context, responderType alert.ResponderType, parameter string) []alert.Responder {
	if val, success := getVal(parameter, c); success {
		responderNames := strings.Split(val, ",")

		var responders []alert.Responder

		for _, name := range responderNames {
			responders = append(responders, alert.Responder{
				Name:     name,
				Username: name,
				Type:     responderType,
			})
		}
		return responders
	}
	return nil
}

func extractDetailsFromCommand(c *gcli.Context) map[string]string {
	details := make(map[string]string)
	extraProps := c.StringSlice("D")
	for i := 0; i < len(extraProps); i++ {
		prop := extraProps[i]
		if !isEmpty("D", prop, c) && strings.Contains(prop, "=") {
			p := strings.Split(prop, "=")
			details[p[0]] = strings.Join(p[1:], "=")
		} else {
			printMessage(ERROR, "Dynamic parameters should have the value of the form a=b, but got: " + prop + "\n")
			gcli.ShowCommandHelp(c, c.Command.Name)
			os.Exit(1)
		}
	}

	return details
}

// GetAlertAction retrieves specified alert details from Opsgenie.
func GetAlertAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}
	req := alert.GetAlertRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	printMessage(DEBUG,"Get alert request prepared from flags, sending request to Opsgenie...")

	resp, err := cli.Get(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	outputFormat := strings.ToLower(c.String("output-format"))
	printMessage(DEBUG,"Got Alert successfully, and will print as " + outputFormat)
	switch outputFormat {
	case "yaml":
		output, err := resultToYAML(resp)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	default:
		isPretty := c.IsSet("pretty")
		output, err := resultToJSON(resp, isPretty)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	}
}

// AttachFileAction attaches a file to an alert at Opsgenie.
func AttachFileAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.CreateAlertAttachmentRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)
	if val, success := getVal("filePath", c); success {
		req.FilePath = val
	}
	if val, success := getVal("fileName", c); success {
		req.FileName = val
	}
	if val, success := getVal("indexFile", c); success {
		req.IndexFile = val
	}

	req.User = grabUsername(c)

	printMessage(DEBUG,"Attach request prepared from flags, sending request to Opsgenie..")

	response, err := cli.CreateAlertAttachments(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	printMessage(DEBUG,"File attached to alert successfully.")
	printMessage(INFO, "Result : " + response.Result + "\n")
}

// GetAttachmentAction retrieves a download link to specified alert attachment
func GetAttachmentAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)

	if err != nil {
		os.Exit(1)
	}

	req := alert.GetAttachmentRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("attachmentId", c); success {
		req.AttachmentId = val
	}

	printMessage(DEBUG,"Get alert attachment request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.GetAlertAttachment(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	printMessage(DEBUG,"Got Alert Attachment successfully, and will print download link.")
	printMessage(INFO, "Download Link: " + resp.Url)
}

// DownloadAttachmentAction downloads the attachment specified with attachmentId for given alert
func DownloadAttachmentAction(c *gcli.Context) {
	var destinationPath string
	cli, err := NewAlertClient(c)

	if err != nil {
		os.Exit(1)
	}

	req := alert.GetAttachmentRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("attachmentId", c); success {
		req.AttachmentId = val
	}

	if val, success := getVal("destinationPath", c); success {
		destinationPath = val
	}

	printMessage(DEBUG,"Download alert attachment request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.GetAlertAttachment(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	fileName := resp.Name
	downloadLink := resp.Url

	var output *os.File

	if destinationPath != "" {
		output, err = os.Create(destinationPath + "/" + fileName)
	} else {
		output, err = os.Create(fileName)
	}

	if err != nil {
		printMessage(ERROR, "Error while creating " + fileName + "-" + err.Error())
		return
	}
	defer output.Close()

	response, err := http.Get(downloadLink)
	if err != nil {
		printMessage(ERROR, "Error while downloading " + fileName + "-" + err.Error())
		return
	}
	defer response.Body.Close()

	_, err = io.Copy(output, response.Body)

	if err != nil {
		printMessage(ERROR,"Error while downloading " + fileName + " - " + err.Error())
		return
	}
}

// ListAlertAttachmentsAction returns a list of attachment meta information for specified alert
func ListAlertAttachmentsAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)

	if err != nil {
		os.Exit(1)
	}

	req := alert.ListAttachmentsRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	printMessage(DEBUG,"List alert attachments request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.ListAlertsAttachments(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	outputFormat := strings.ToLower(c.String("output-format"))
	printMessage(DEBUG,"List Alert Attachment successfully, and will print as " + outputFormat)
	switch outputFormat {
	case "yaml":
		output, err := resultToYAML(resp.Attachment)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	default:
		isPretty := c.IsSet("pretty")
		output, err := resultToJSON(resp.Attachment, isPretty)

		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}

		printMessage(INFO, output)
	}
}

// DeleteAlertAttachmentAction deletes the specified alert attachment from alert
func DeleteAlertAttachmentAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.DeleteAttachmentRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("attachmentId", c); success {
		req.AttachmentId = val
	}

	printMessage(DEBUG,"Delete alert attachment request prepared from flags, sending request to OpsGenie..")

	resp, err := cli.DeleteAlertAttachment(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	printMessage(DEBUG,"Alert attachment will be deleted. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
	printMessage(INFO,"Result: " + resp.Result)
}

// AcknowledgeAction acknowledges an alert at Opsgenie.
func AcknowledgeAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.AcknowledgeAlertRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Acknowledge alert request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.Acknowledge(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	printMessage(DEBUG,"Acknowledge request will be processed. RequestID " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// AssignOwnerAction assigns the specified user as the owner of the alert at Opsgenie.
func AssignOwnerAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.AssignRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("owner", c); success {
		req.Owner = alert.User{Username: val}
	}
	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Assign ownership request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.AssignAlert(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	printMessage(DEBUG,"Ownership assignment request will be processed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// AddTeamAction adds a team to an alert at Opsgenie.
func AddTeamAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}
	req := alert.AddTeamRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("team", c); success {
		req.Team = alert.Team{Name: val}
	}
	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Add team request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.AddTeam(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Add team request will be processed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// AddResponderAction adds responder to an alert at Opsgenie.
func AddResponderAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}
	req := alert.AddResponderRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if valType, success := getVal("type", c); success {
		if val, success := getVal("responder", c); success {
			req.Responder = alert.Responder{
				Type:     alert.ResponderType(valType),
				Name:     val,
				Username: val,
			}
		}
	}
	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Add responder request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.AddResponder(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Add responder request will be processed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// AddTagsAction adds tags to an alert at Opsgenie.
func AddTagsAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.AddTagsRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)
	if val, success := getVal("tags", c); success {
		req.Tags = strings.Split(val, ",")
	}
	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Add tag request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.AddTags(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Add tags request will be processed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// AddNoteAction adds a note to an alert at Opsgenie.
func AddNoteAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.AddNoteRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Add note request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.AddNote(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Add note request will be processed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// ExecuteActionAction executes a custom action on an alert at Opsgenie.
func ExecuteActionAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.ExecuteCustomActionAlertRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if action, success := getVal("action", c); success {
		req.Action = action
	}
	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Execute action request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.ExecuteCustomAction(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Execute custom action request will be processed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// CloseAlertAction closes an alert at Opsgenie.
func CloseAlertAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.CloseAlertRequest{}
	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)
	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Close alert request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.Close(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Alert will be closed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// DeleteAlertAction deletes an alert at Opsgenie.
func DeleteAlertAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.DeleteAlertRequest{}
	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}

	printMessage(DEBUG,"Delete alert request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.Delete(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	printMessage(DEBUG,"Alert will be deleted. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// ListAlertsAction retrieves alert details from Opsgenie.
func ListAlertsAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := generateListAlertRequest(c)

	printMessage(DEBUG,"List alerts request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.List(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	outputFormat := strings.ToLower(c.String("output-format"))
	printMessage(DEBUG,"Got Alerts successfully, and will print as " + outputFormat)
	switch outputFormat {
	case "yaml":
		output, err := resultToYAML(resp.Alerts)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	default:
		isPretty := c.IsSet("pretty")
		output, err := resultToJSON(resp.Alerts, isPretty)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO,output)
	}
}

func generateListAlertRequest(c *gcli.Context) alert.ListAlertRequest {
	req := alert.ListAlertRequest{}

	if val, success := getVal("limit", c); success {
		limit, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			os.Exit(2)
		}
		req.Limit = int(limit)
	}
	if val, success := getVal("sort", c); success {
		req.Sort = alert.SortField(val)
	}
	if val, success := getVal("order", c); success {
		req.Order = alert.Order(val)
	}
	if val, success := getVal("searchIdentifier", c); success {
		req.SearchIdentifier = val
	}

	if val, success := getVal("searchIdentifierType", c); success {
		if alert.SearchIdentifierType(val) == alert.NAME {
			req.SearchIdentifierType = alert.NAME
		} else {
			req.SearchIdentifierType = alert.ID

		}
	}

	if val, success := getVal("offset", c); success {
		offset, err := strconv.Atoi(val)
		if err != nil {
			os.Exit(2)
		}
		req.Offset = offset
	}

	if val, success := getVal("query", c); success {
		req.Query = val
	} else {
		generateQueryUsingOldStyleParams(c, &req)
	}
	return req
}

func generateQueryUsingOldStyleParams(c *gcli.Context, req *alert.ListAlertRequest) {
	var queries []string
	if val, success := getVal("createdAfter", c); success {
		createdAfter, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			os.Exit(2)
		}
		queries = append(queries, "createdAt > "+strconv.FormatUint(createdAfter, 10))
	}
	if val, success := getVal("createdBefore", c); success {
		createdBefore, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			os.Exit(2)
		}
		queries = append(queries, "createdAt < "+strconv.FormatUint(createdBefore, 10))
	}
	if val, success := getVal("updatedAfter", c); success {
		updatedAfter, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			os.Exit(2)
		}
		queries = append(queries, "updatedAt > "+strconv.FormatUint(updatedAfter, 10))
	}
	if val, success := getVal("updatedBefore", c); success {
		updatedBefore, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			os.Exit(2)
		}
		queries = append(queries, "updatedAt < "+strconv.FormatUint(updatedBefore, 10))
	}
	if val, success := getVal("status", c); success {
		queries = append(queries, "status: "+val)
	}
	if val, success := getVal("teams", c); success {
		for _, teamName := range strings.Split(val, ",") {
			queries = append(queries, "teams: "+teamName)

		}
	}
	if val, success := getVal("tags", c); success {
		var tags []string
		operator := "AND"

		if val, success := getVal("tagsOperator", c); success {
			operator = val
		}

		for _, tag := range strings.Split(val, ",") {
			tags = append(tags, tag)
		}

		tagsPart := "tag: (" + strings.Join(tags, " "+operator+" ") + ")"
		queries = append(queries, tagsPart)
	}
	if len(queries) != 0 {
		req.Query = strings.Join(queries, " AND ")
	}
}

// CountAlertsAction retrieves number of alerts from Opsgenie.
func CountAlertsAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}
	req := generateListAlertRequest(c)

	printMessage(DEBUG,"Count alerts request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.List(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(INFO, fmt.Sprint(len(resp.Alerts)))
}

// ListAlertNotesAction retrieves specified alert notes from Opsgenie.
func ListAlertNotesAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.ListAlertNotesRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("limit", c); success {
		limit, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			os.Exit(2)
		}
		req.Limit = uint32(limit)
	}

	if val, success := getVal("order", c); success {
		req.Order = alert.Order(val)
	}

	if val, success := getVal("direction", c); success {
		req.Direction = alert.RequestDirection(val)
	}

	if val, success := getVal("offset", c); success {
		req.Offset = val
	}

	printMessage(DEBUG,"List alert notes request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.ListAlertNotes(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	outputFormat := strings.ToLower(c.String("output-format"))
	printMessage(DEBUG,"Alert notes listed successfully, and will print as " + outputFormat)
	switch outputFormat {
	case "yaml":
		output, err := resultToYAML(resp.AlertLog)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	default:
		isPretty := c.IsSet("pretty")
		output, err := resultToJSON(resp.AlertLog, isPretty)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	}
}

// ListAlertLogsAction retrieves specified alert logs from Opsgenie.
func ListAlertLogsAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}
	req := alert.ListAlertLogsRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("limit", c); success {
		limit, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			os.Exit(2)
		}
		req.Limit = uint32(limit)
	}

	if val, success := getVal("order", c); success {
		req.Order = alert.Order(val)
	}

	if val, success := getVal("direction", c); success {
		req.Direction = alert.RequestDirection(val)
	}

	if val, success := getVal("offset", c); success {
		req.Offset = val
	}

	printMessage(DEBUG,"List alert notes request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.ListAlertLogs(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	outputFormat := strings.ToLower(c.String("output-format"))
	printMessage(DEBUG,"Alert notes listed successfully, and will print as " + outputFormat)
	switch outputFormat {
	case "yaml":
		output, err := resultToYAML(resp.AlertLog)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	default:
		isPretty := c.IsSet("pretty")
		output, err := resultToJSON(resp.AlertLog, isPretty)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	}
}

// ListAlertRecipientsAction retrieves specified alert recipients from Opsgenie.
func ListAlertRecipientsAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}
	req := alert.ListAlertRecipientRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	printMessage(DEBUG,"List alert recipients request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.ListAlertRecipients(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	outputFormat := strings.ToLower(c.String("output-format"))
	printMessage(DEBUG,"Alert recipients listed successfully, and will print as " + outputFormat)
	switch outputFormat {
	case "yaml":
		output, err := resultToYAML(resp.AlertRecipients)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	default:
		isPretty := c.IsSet("pretty")
		output, err := resultToJSON(resp.AlertRecipients, isPretty)
		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}
		printMessage(INFO, output)
	}
}

// UnAcknowledgeAction unAcknowledges an alert at Opsgenie.
func UnAcknowledgeAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.UnacknowledgeAlertRequest{}
	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"UnAcknowledge alert request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.Unacknowledge(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}

	printMessage(DEBUG,"Alert will be unAcknowledged. RequestID: " + resp.RequestId)
	printMessage(INFO, "RequestID: " + resp.RequestId)
}

// SnoozeAction snoozes an alert at Opsgenie.
func SnoozeAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.SnoozeAlertRequest{}
	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	if val, success := getVal("endDate", c); success {

		endTime, err := time.Parse(time.RFC3339, val)

		if err != nil {
			printMessage(ERROR,err.Error())
			os.Exit(1)
		}

		req.EndTime = endTime
	}
	printMessage(DEBUG,"Snooze request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.Snooze(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"will be snoozed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// RemoveTagsAction removes tags from an alert at Opsgenie.
func RemoveTagsAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.RemoveTagsRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("tags", c); success {
		req.Tags = val
	}
	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Remove tags request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.RemoveTags(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Tags will be removed. RequestID: " + resp.RequestId)
	printMessage(INFO, "RequestID: " + resp.RequestId)
}

// AddDetailsAction adds details to an alert at Opsgenie.
func AddDetailsAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.AddDetailsRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}
	if c.IsSet("D") {
		req.Details = extractDetailsFromCommand(c)
	}
	printMessage(DEBUG,"Add details request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.AddDetails(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Details will be added. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// RemoveDetailsAction removes details from an alert at Opsgenie.
func RemoveDetailsAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.RemoveDetailsRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("keys", c); success {
		req.Keys = val
	}
	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Remove details request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.RemoveDetails(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Details will be removed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

// EscalateToNextAction processes the next available rule in the specified escalation.
func EscalateToNextAction(c *gcli.Context) {
	cli, err := NewAlertClient(c)
	if err != nil {
		os.Exit(1)
	}

	req := alert.EscalateToNextRequest{}

	if val, success := getVal("id", c); success {
		req.IdentifierValue = val
	}
	req.IdentifierType = grabIdentifierType(c)

	if val, success := getVal("escalationId", c); success {
		req.Escalation.ID = val
	}
	if val, success := getVal("escalationName", c); success {
		req.Escalation.Name = val
	}
	req.User = grabUsername(c)
	if val, success := getVal("source", c); success {
		req.Source = val
	}
	if val, success := getVal("note", c); success {
		req.Note = val
	}

	printMessage(DEBUG,"Escalate to next request prepared from flags, sending request to Opsgenie..")

	resp, err := cli.EscalateToNext(nil, &req)
	if err != nil {
		printMessage(ERROR,err.Error())
		os.Exit(1)
	}
	printMessage(DEBUG,"Escalated to next request will be processed. RequestID: " + resp.RequestId)
	printMessage(INFO,"RequestID: " + resp.RequestId)
}

func grabIdentifierType(c *gcli.Context) alert.AlertIdentifier {
	if val, success := getVal("identifier", c); success {
		if val == "tiny" {
			return alert.TINYID
		} else if val == "alias" {
			return alert.ALIAS
		}
	}
	return alert.ALERTID
}
