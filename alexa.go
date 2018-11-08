package alexa

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"
)

const sdkVersion = "1.0"
const launchRequestName = "LaunchRequest"
const intentRequestName = "IntentRequest"
const sessionEndedRequestName = "SessionEndedRequest"

var timestampTolerance = 150

// Built-in intents
const (
	//HelpIntent is the Alexa built-in Help Intent
	HelpIntent = "AMAZON.HelpIntent"

	//CancelIntent is the Alexa built-in Cancel Intent
	CancelIntent = "AMAZON.CancelIntent"

	//StopIntent is the Alexa built-in Stop Intent
	StopIntent = "AMAZON.StopIntent"

	PauseIntent = "AMAZON.PauseIntent"

	StartOverIntent = "AMAZON.StartOverIntent"

	RepeatIntent = "AMAZON.RepeatIntent"
)

// Locales
const (
	// LocaleItalian is the locale for Italian
	LocaleItalian = "it-IT"

	// LocaleGerman is the locale for standard dialect German
	LocaleGerman = "de-DE"

	// LocaleAustralianEnglish is the locale for Australian English
	LocaleAustralianEnglish = "en-AU"

	// LocaleCanadianEnglish is the locale for Canadian English
	LocaleCanadianEnglish = "en-CA"

	// LocaleBritishEnglish is the locale for UK English
	LocaleBritishEnglish = "en-GB"

	// LocaleIndianEnglish is the locale for Indian English
	LocaleIndianEnglish = "en-IN"

	// LocaleAmericanEnglish is the locale for American English
	LocaleAmericanEnglish = "en-US"

	// LocaleJapanese is the locale for Japanese
	LocaleJapanese = "ja-JP"
)

func IsEnglish(locale string) bool {
	return strings.HasPrefix(locale, "en-")
}

// ErrRequestEnvelopeNil reports that the request envelope was nil
// there might be edge case which causes panic if for whatever reason this object is empty
var ErrRequestEnvelopeNil = errors.New("request envelope was nil")

// Alexa defines the primary interface to use to create an Alexa request handler.
type Alexa struct {
	ApplicationID       string
	RequestHandler      RequestHandler
	IgnoreApplicationID bool
	IgnoreTimestamp     bool
}

// RequestHandler defines the interface that must be implemented to handle
// Alexa Requests
type RequestHandler interface {
	OnSessionStarted(context.Context, *Request, *Session, *Context, *Response) error
	OnLaunch(context.Context, *Request, *Session, *Context, *Response) error
	OnIntent(context.Context, *Request, *Session, *Context, *Response) error
	OnSessionEnded(context.Context, *Request, *Session, *Context, *Response) error
}

// RequestEnvelope contains the data passed from Alexa to the request handler.
type RequestEnvelope struct {
	Version string   `json:"version"`
	Session *Session `json:"session"`
	Request *Request `json:"request"`
	Context *Context `json:"context"`
}

// Session contains the session data from the Alexa request.
type Session struct {
	New         bool   `json:"new"`
	SessionID   string `json:"sessionId"`
	Application struct {
		ApplicationID string `json:"applicationId"`
	} `json:"application"`
	Attributes map[string]interface{} `json:"attributes"`
	User       struct {
		UserID      string `json:"userId"`
		AccessToken string `json:"accessToken,omitempty"`
	} `json:"user"`
}

type PlayerActivity string

const (
	PlayerActivityIdle           PlayerActivity = "IDLE"            // Nothing was playing, no enqueued items.
	PlayerActivityPaused         PlayerActivity = "PAUSED"          // Stream was paused.
	PlayerActivityPlaying        PlayerActivity = "PLAYING"         // Stream was playing.
	PlayerActivityBufferUnderrun PlayerActivity = "BUFFER_UNDERRUN" // Buffer underrun
	PlayerActivityFinished       PlayerActivity = "FINISHED"        // Stream was finished playing.
	PlayerActivityStopped        PlayerActivity = "STOPPED"         // Stream was interrupted.
)

// Context contains the context data from the Alexa Request.
type Context struct {
	AudioPlayer struct {
		Token                string         `json:"token"`
		OffsetInMilliseconds int64          `json:"offsetInMilliseconds"`
		PlayerActivity       PlayerActivity `json:"playerActivity"`
	} `json:"AudioPlayer"`

	Display struct {
		Token string `json:"token"`
	} `json:"Display"`

	System struct {
		Application struct {
			ApplicationID string `json:"applicationId"`
		} `json:"application"`
		User struct {
			UserID      string `json:"userId"`
			AccessToken string `json:"accessToken"`
		} `json:"user"`
		Device struct {
			DeviceID            string `json:"deviceId"`
			SupportedInterfaces struct {
				AudioPlayer struct {
				} `json:"AudioPlayer"`
				Display struct {
					TemplateVersion string `json:"templateVersion"`
					MarkupVersion   string `json:"markupVersion"`
				} `json:"Display"`
			} `json:"supportedInterfaces"`
		} `json:"device"`
		APIEndpoint    string `json:"apiEndpoint"`
		APIAccessToken string `json:"apiAccessToken"`
	} `json:"System"`
}

// Request contains the data in the request within the main request.
type Request struct {
	Locale      string `json:"locale,omitempty"`
	Timestamp   string `json:"timestamp"`
	Type        string `json:"type"`
	RequestID   string `json:"requestId"`
	DialogState string `json:"dialogState,omitempty"`
	Intent      Intent `json:"intent,omitempty"`
	Name        string `json:"name"`
	Reason      string `json:"reason,omitempty"`
}

// GetSessionID is a convenience method for getting the session ID out of a Request.
func (r *RequestEnvelope) GetSessionID() string {
	return r.Session.SessionID
}

// GetUserID is a convenience method for getting the user identifier out of a Request.
func (r *RequestEnvelope) GetUserID() string {
	return r.Session.User.UserID
}

// GetRequestType is a convenience method for getting the request type out of a Request.
func (r *RequestEnvelope) GetRequestType() string {
	return r.Request.Type
}

// GetIntentName is a convenience method for getting the intent name out of a Request.
func (r *RequestEnvelope) GetIntentName() string {
	if r.GetRequestType() == "IntentRequest" {
		return r.Request.Intent.Name
	}

	return r.GetRequestType()
}

// GetSlotValue is a convenience method for getting the value of the specified slot out of a Request
// as a string. An error is returned if a slot with that value is not found in the request.
func (r *RequestEnvelope) GetSlotValue(slotName string) (string, error) {
	slot, err := r.GetSlot(slotName)

	if err != nil {
		return "", err
	}

	return slot.Value, nil
}

// GetSlot will return an IntentSlot from the Request with the given name.
func (r *RequestEnvelope) GetSlot(slotName string) (*IntentSlot, error) {
	if slot, ok := r.Request.Intent.Slots[slotName]; ok {
		return &slot, nil
	}

	return nil, errors.New("slot name not found")
}

// AllSlots will return a map of all the slots in the Request mapped by their name.
func (r *RequestEnvelope) AllSlots() map[string]IntentSlot {
	return r.Request.Intent.Slots
}

// Locale returns the locale specified in the request.
func (r *RequestEnvelope) Locale() string {
	return r.Request.Locale
}

// Intent contains the data about the Alexa Intent requested.
type Intent struct {
	Name               string                `json:"name"`
	ConfirmationStatus ConfirmationStatus    `json:"confirmationStatus,omitempty"`
	Slots              map[string]IntentSlot `json:"slots"`
}

// IntentSlot contains the data for one Slot
type IntentSlot struct {
	Name               string             `json:"name"`
	ConfirmationStatus ConfirmationStatus `json:"confirmationStatus,omitempty"`
	Value              string             `json:"value"`
	Resolutions        *Resolutions       `json:"resolutions,omitempty"`
}

// ConfirmationStatus represents the status of either a dialog or slot confirmation.
type ConfirmationStatus string

const (
	// ConfConfirmed indicates the intent or slot has been confirmed by the end user.
	ConfConfirmed ConfirmationStatus = "CONFIRMED"

	// ConfDenied means the end user indicated the intent or slot should NOT proceed.
	ConfDenied ConfirmationStatus = "DENIED"

	// ConfNone means there has been not acceptance or denial of the intent or slot.
	ConfNone ConfirmationStatus = "NONE"
)

// Resolutions contain the (optional) ID of a slot
type Resolutions struct {
	ResolutionsPerAuthority []EchoResolutionPerAuthority `json:"resolutionsPerAuthority"`
}

// EchoResolutionPerAuthority contains information about a single slot resolution from a single
// authority. The values silce will contain all possible matches for different slots.
// These resolutions are most interesting when working with synonyms.
type EchoResolutionPerAuthority struct {
	Authority string `json:"authority"`
	Status    struct {
		Code string `json:"code"`
	} `json:"status"`
	Values []map[string]struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"values"`
}

// ResponseEnvelope contains the Response and additional attributes.
type ResponseEnvelope struct {
	Version           string                 `json:"version"`
	SessionAttributes map[string]interface{} `json:"sessionAttributes,omitempty"`
	Response          *Response              `json:"response"`
}

// Response contains the body of the response.
type Response struct {
	OutputSpeech     *OutputSpeech `json:"outputSpeech,omitempty"`
	Card             *Card         `json:"card,omitempty"`
	Reprompt         *Reprompt     `json:"reprompt,omitempty"`
	Directives       []interface{} `json:"directives,omitempty"`
	ShouldEndSession bool          `json:"shouldEndSession"`
}

// OutputSpeech contains the data the defines what Alexa should say to the user.
type OutputSpeech struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	SSML         string `json:"ssml,omitempty"`
	PlayBehavior string `json:"playBehavior,omitempty"`
}

type PlayBehavior string

const (
	PlayBehaviorEnqueue         PlayBehavior = "ENQUEUE"          // Add this speech to the end of the queue. Do not interrupt Alexa's current speech. This is the default value for all skills that do not use the GameEngine interface.
	PlayBehaviorReplaceAll      PlayBehavior = "REPLACE_ALL"      // Immediately begin playback of this speech, and replace any current and enqueued speech. This is the default value for all skills that use the GameEngine interface.
	PlayBehaviorReplaceEnqueued PlayBehavior = "REPLACE_ENQUEUED" // Replace all speech in the queue with this speech. Do not interrupt Alexa's current speech.
)

// Card contains the data displayed to the user by the Alexa app.
type Card struct {
	Type    CardType `json:"type"`
	Title   string   `json:"title,omitempty"`
	Content string   `json:"content,omitempty"`
	Text    string   `json:"text,omitempty"`
	Image   *Image   `json:"image,omitempty"`
}

type CardType string

const (
	CardTypeSimple                   CardType = "Simple"                   // A card that contains a title and plain text content.
	CardTypeStandard                 CardType = "Standard"                 // A card that contains a title, text content, and an image to display.
	CardTypeLinkAccount              CardType = "LinkAccount"              // A card that displays a link to an authorization URI that the user can use to link their Alexa account with a user in another system. See Account Linking for Custom Skills for details.
	CardTypeAskForPermissionsConsent CardType = "AskForPermissionsConsent" // A card that asks the customer for consent to obtain specific customer information, such as Alexa lists or address information. See Alexa Shopping and To-Do Lists and Enhance Your Skill with Address Information.
)

// Image provides URL(s) to the image to display in resposne to the request.
type Image struct {
	SmallImageURL string `json:"smallImageUrl,omitempty"`
	LargeImageURL string `json:"largeImageUrl,omitempty"`
}

// Reprompt contains data about whether Alexa should prompt the user for more data.
type Reprompt struct {
	OutputSpeech *OutputSpeech `json:"outputSpeech,omitempty"`
}

// AudioPlayerDirective contains device level instructions on how to handle the response.
type AudioPlayerDirective struct {
	Type         string     `json:"type"`
	PlayBehavior string     `json:"playBehavior,omitempty"`
	AudioItem    *AudioItem `json:"audioItem,omitempty"`
}

// AudioItem contains an audio Stream definition for playback.
type AudioItem struct {
	Stream Stream `json:"stream,omitempty"`
}

// Stream contains instructions on playing an audio stream.
type Stream struct {
	Token                string `json:"token"`
	URL                  string `json:"url"`
	OffsetInMilliseconds int    `json:"offsetInMilliseconds"`
}

// DialogDirective contains directives for use in Dialog prompts.
type DialogDirective struct {
	Type          string  `json:"type"`
	SlotToElicit  string  `json:"slotToElicit,omitempty"`
	SlotToConfirm string  `json:"slotToConfirm,omitempty"`
	UpdatedIntent *Intent `json:"updatedIntent,omitempty"`
}

// ProcessRequest handles a request passed from Alexa
func (alexa *Alexa) ProcessRequest(ctx context.Context, requestEnv *RequestEnvelope) (*ResponseEnvelope, error) {
	if requestEnv == nil {
		return nil, ErrRequestEnvelopeNil
	}

	if !alexa.IgnoreApplicationID {
		err := alexa.verifyApplicationID(requestEnv)
		if err != nil {
			return nil, err
		}
	}
	if !alexa.IgnoreTimestamp {
		err := alexa.verifyTimestamp(requestEnv)
		if err != nil {
			return nil, err
		}
	} else {
		log.Println("Ignoring timestamp verification.")
	}

	request := requestEnv.Request
	session := requestEnv.Session
	if session.Attributes == nil {
		session.Attributes = make(map[string]interface{})
	}
	context := requestEnv.Context

	responseEnv := &ResponseEnvelope{}
	responseEnv.Version = sdkVersion
	responseEnv.Response = &Response{}
	responseEnv.Response.ShouldEndSession = true // Set default value.

	response := responseEnv.Response

	// If it is a new session, invoke onSessionStarted
	if session.New {
		err := alexa.RequestHandler.OnSessionStarted(ctx, request, session, context, response)
		if err != nil {
			log.Println("Error handling OnSessionStarted.", err.Error())
			return nil, err
		}
	}

	switch requestEnv.Request.Type {
	case launchRequestName:
		err := alexa.RequestHandler.OnLaunch(ctx, request, session, context, response)
		if err != nil {
			log.Println("Error handling OnLaunch.", err.Error())
			return nil, err
		}
	case intentRequestName:
		err := alexa.RequestHandler.OnIntent(ctx, request, session, context, response)
		if err != nil {
			log.Println("Error handling OnIntent.", err.Error())
			return nil, err
		}
	case sessionEndedRequestName:
		err := alexa.RequestHandler.OnSessionEnded(ctx, request, session, context, response)
		if err != nil {
			log.Println("Error handling OnSessionEnded.", err.Error())
			return nil, err
		}
	}

	// Copy Session Attributes into ResponseEnvelope
	responseEnv.SessionAttributes = make(map[string]interface{})
	for n, v := range session.Attributes {
		fmt.Println("Setting ", n, "to", v)
		responseEnv.SessionAttributes[n] = v
	}

	return responseEnv, nil
}

// SetTimestampTolerance sets the maximum number of seconds to allow between
// the current time and the request Timestamp.  Default value is 150 seconds.
func (alexa *Alexa) SetTimestampTolerance(seconds int) {
	timestampTolerance = seconds
}

// SetSimpleCard creates a new simple card with the specified content.
func (r *Response) SetSimpleCard(title string, content string) {
	r.Card = &Card{Type: "Simple", Title: title, Content: content}
}

// SetStandardCard creates a new standard card with the specified content.
func (r *Response) SetStandardCard(title string, text string, smallImageURL string, largeImageURL string) {
	r.Card = &Card{Type: "Standard", Title: title, Text: text}
	r.Card.Image = &Image{SmallImageURL: smallImageURL, LargeImageURL: largeImageURL}
}

// SetLinkAccountCard creates a new LinkAccount card.
func (r *Response) SetLinkAccountCard() {
	r.Card = &Card{Type: "LinkAccount"}
}

// SetOutputSpeech sets the OutputSpeech type to text and sets the value specified.
func (r *Response) SetOutputSpeech(text string) {
	r.OutputSpeech = &OutputSpeech{Type: "PlainText", Text: text}
}

// SetOutputSSML sets the OutputSpeech type to ssml and sets the value specified.
func (r *Response) SetOutputSSML(ssml string) {
	r.OutputSpeech = &OutputSpeech{Type: "SSML", SSML: ssml}
}

// SetRepromptText created a Reprompt if needed and sets the OutputSpeech type to text and sets the value specified.
func (r *Response) SetRepromptText(text string) {
	if r.Reprompt == nil {
		r.Reprompt = &Reprompt{}
	}
	r.Reprompt.OutputSpeech = &OutputSpeech{Type: "PlainText", Text: text}
}

// SetRepromptSSML created a Reprompt if needed and sets the OutputSpeech type to ssml and sets the value specified.
func (r *Response) SetRepromptSSML(ssml string) {
	if r.Reprompt == nil {
		r.Reprompt = &Reprompt{}
	}
	r.Reprompt.OutputSpeech = &OutputSpeech{Type: "SSML", SSML: ssml}
}

// SetEndSession is a convenience method for setting the flag in the response that will
// indicate if the session between the end user's device and the skillserver should be closed.
func (r *Response) SetEndSession(flag bool) *Response {
	r.ShouldEndSession = flag

	return r
}

// AddAudioPlayer adds an AudioPlayer directive to the Response.
func (r *Response) AddAudioPlayer(playerType, playBehavior, streamToken, url string, offsetInMilliseconds int) {
	d := AudioPlayerDirective{
		Type:         playerType,
		PlayBehavior: playBehavior,
		AudioItem: &AudioItem{
			Stream: Stream{
				Token:                streamToken,
				URL:                  url,
				OffsetInMilliseconds: offsetInMilliseconds,
			},
		},
	}
	r.Directives = append(r.Directives, d)
}

// AddDialogDirective adds a Dialog directive to the Response.
func (r *Response) AddDialogDirective(dialogType, slotToElicit, slotToConfirm string, intent *Intent) {
	d := DialogDirective{
		Type:          dialogType,
		SlotToElicit:  slotToElicit,
		SlotToConfirm: slotToConfirm,
		UpdatedIntent: intent,
	}
	r.Directives = append(r.Directives, d)
}

// verifyApplicationId verifies that the ApplicationID sent in the request
// matches the one configured for this skill.
func (alexa *Alexa) verifyApplicationID(request *RequestEnvelope) error {
	if request == nil {
		return ErrRequestEnvelopeNil
	}

	appID := alexa.ApplicationID
	requestAppID := request.Session.Application.ApplicationID
	if appID == "" {
		return errors.New("application ID was set to an empty string")
	}
	if requestAppID == "" {
		return errors.New("request Application ID was set to an empty string")
	}
	if appID != requestAppID {
		return errors.New("request Application ID does not match expected ApplicationId")
	}

	return nil
}

// verifyTimestamp compares the request timestamp to the current timestamp
// and returns an error if they are too far apart.
func (alexa *Alexa) verifyTimestamp(request *RequestEnvelope) error {
	if request == nil {
		return ErrRequestEnvelopeNil
	}

	timestamp, err := time.Parse(time.RFC3339, request.Request.Timestamp)
	if err != nil {
		return errors.New("unable to parse request timestamp.  Err: " + err.Error())
	}
	now := time.Now()
	delta := now.Sub(timestamp)
	deltaSecsAbs := math.Abs(delta.Seconds())
	if deltaSecsAbs > float64(timestampTolerance) {
		return errors.New("invalid Timestamp. The request timestamp " + timestamp.String() + " was off the current time " + now.String() + " by more than " + strconv.FormatInt(int64(timestampTolerance), 10) + " seconds.")
	}

	return nil
}
