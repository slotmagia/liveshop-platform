package modulemanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

const APIVersion = "liveshop.io/v1"

// KindModuleRelease is the only manifest kind the registry accepts.
const KindModuleRelease = "ModuleRelease"

// NewRelease assembles the canonical manifest document of one module release.
// Callers that republish a stored release must build it here so the apiVersion
// and kind are never spelled out by a transport.
func NewRelease(metadata Metadata, spec Spec) Manifest {
	return Manifest{APIVersion: APIVersion, Kind: KindModuleRelease, Metadata: metadata, Spec: spec}
}

var validSurfaces = map[string]bool{"admin": true, "merch": true, "shop": true, "live": true, "internal": true}
var validKinds = map[string]bool{"page": true, "slot": true, "widget": true, "action": true}
var validArtifactTypes = map[string]bool{"iframe": true, "remote-esm": true}
var validMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}
var validAuthentication = map[string]bool{"module-session": true, "guest-session": true, "workload-identity": true, "public": true}
var validIdempotency = map[string]bool{"safe": true, "idempotent": true, "non-idempotent": true}
var validFieldLocations = map[string]bool{"path": true, "query": true, "header": true, "body": true, "context": true}
var validFieldTypes = map[string]bool{"string": true, "integer": true, "number": true, "boolean": true, "object": true, "array": true, "bytes": true}
var validGRPCInvocations = map[string]bool{"unary": true, "server-stream": true, "client-stream": true, "bidi-stream": true}
var validFrontendInvocations = map[string]bool{"http": true, "host-event": true, "navigation": true, "module-export": true}
var moduleIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
var semverPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
var permissionPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*){2,}$`)
var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var eventKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-zA-Z0-9]*){2,}$`)
var notifyVariablePattern = regexp.MustCompile(`^[a-z][a-zA-Z0-9]{0,31}$`)
var validNotifyChannels = map[string]bool{"SMS": true, "EMAIL": true, "IN_APP": true}
var validNotifyDispatch = map[string]bool{"SYNC": true, "ASYNC": true, "SCHEDULED": true}

type Manifest struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Spec       Spec     `json:"spec"`
}

type Metadata struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Spec struct {
	Backend       Backend                `json:"backend"`
	Permissions   []PermissionDefinition `json:"permissions"`
	Contributions []Contribution         `json:"contributions"`
}

// PermissionDefinition declares a capability owned by a module. It never
// grants that capability to a user; grants are owned by Identity IAM.
type PermissionDefinition struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description,omitempty"`
}

type Backend struct {
	Service    string      `json:"service"`
	Origin     string      `json:"origin"`
	HTTPRoutes []HTTPRoute `json:"httpRoutes"`
	GRPC       *GRPC       `json:"grpc,omitempty"`
}

type HTTPRoute struct {
	Surface    string          `json:"surface"`
	Prefix     string          `json:"prefix"`
	Operations []HTTPOperation `json:"operations"`
}

type GRPC struct {
	Service           string       `json:"service"`
	ContractVersion   string       `json:"contractVersion"`
	Endpoint          string       `json:"endpoint"`
	TransportSecurity string       `json:"transportSecurity"`
	Methods           []GRPCMethod `json:"methods"`
}

// CapabilityField is a transport-neutral, machine-readable field description.
// Location is used by HTTP parameters and omitted for gRPC/frontend payloads.
type CapabilityField struct {
	Name        string `json:"name"`
	Location    string `json:"location,omitempty"`
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

type HTTPOperation struct {
	ID                  string                    `json:"id"`
	Method              string                    `json:"method"`
	Path                string                    `json:"path"`
	Summary             string                    `json:"summary"`
	Description         string                    `json:"description"`
	Authentication      string                    `json:"authentication"`
	Idempotency         string                    `json:"idempotency"`
	RequiredPermissions []string                  `json:"requiredPermissions"`
	RequestFields       []CapabilityField         `json:"requestFields"`
	Responses           []CapabilityResponse      `json:"responses"`
	Notifications       []NotificationDeclaration `json:"notifications,omitempty"`
}

// NotificationDeclaration is a business event a module may Dispatch after its
// own transaction commits. Platform projects the active set; Admin never invents events.
type NotificationDeclaration struct {
	EventKey        string   `json:"eventKey"`
	Title           string   `json:"title"`
	Variables       []string `json:"variables"`
	AllowedChannels []string `json:"allowedChannels"`
	DefaultDispatch string   `json:"defaultDispatch"`
}

type CapabilityResponse struct {
	Status      int               `json:"status"`
	Description string            `json:"description"`
	Fields      []CapabilityField `json:"fields"`
}

type GRPCMethod struct {
	Name                  string            `json:"name"`
	FullMethod            string            `json:"fullMethod"`
	Summary               string            `json:"summary"`
	Description           string            `json:"description"`
	Invocation            string            `json:"invocation"`
	Idempotency           string            `json:"idempotency"`
	RecommendedDeadlineMS int               `json:"recommendedDeadlineMs"`
	RequiredPermissions   []string          `json:"requiredPermissions"`
	RequestFields         []CapabilityField `json:"requestFields"`
	ResponseFields        []CapabilityField `json:"responseFields"`
}

type Contribution struct {
	ID                  string           `json:"id"`
	Surface             string           `json:"surface"`
	Kind                string           `json:"kind"`
	Route               string           `json:"route,omitempty"`
	Outlet              string           `json:"outlet,omitempty"`
	Title               string           `json:"title"`
	Description         string           `json:"description"`
	Icon                string           `json:"icon,omitempty"`
	Sort                int              `json:"sort,omitempty"`
	Navigation          *Navigation      `json:"navigation,omitempty"`
	RequiredPermissions []string         `json:"requiredPermissions"`
	AllowedRoutes       []AllowedRoute   `json:"allowedRoutes"`
	Artifact            Artifact         `json:"artifact"`
	Frontend            FrontendContract `json:"frontend"`
}

// Navigation describes where a page appears in a console menu. It is page
// presentation metadata only; authorization remains entirely permission based.
type Navigation struct {
	GroupID    string `json:"groupId"`
	GroupTitle string `json:"groupTitle"`
	GroupIcon  string `json:"groupIcon,omitempty"`
	GroupSort  int    `json:"groupSort"`
}

type FrontendContract struct {
	Component string            `json:"component"`
	Props     []CapabilityField `json:"props"`
	Events    []FrontendEvent   `json:"events"`
	Actions   []FrontendAction  `json:"actions"`
}

type FrontendEvent struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Payload     []CapabilityField `json:"payload"`
}

type FrontendAction struct {
	ID                  string            `json:"id"`
	Label               string            `json:"label"`
	Description         string            `json:"description"`
	Invocation          string            `json:"invocation"`
	Target              string            `json:"target"`
	Parameters          []CapabilityField `json:"parameters"`
	RequiredPermissions []string          `json:"requiredPermissions"`
}

type AllowedRoute struct {
	Methods             []string `json:"methods"`
	Prefix              string   `json:"prefix"`
	RequiredPermissions []string `json:"requiredPermissions"`
}

type Artifact struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Entry      string `json:"entry"`
	ExportName string `json:"exportName,omitempty"`
	Integrity  string `json:"integrity"`
}

type RuntimeContribution struct {
	ModuleID      string       `json:"moduleId"`
	ModuleVersion string       `json:"moduleVersion"`
	Contribution  Contribution `json:"contribution"`
}

type ActiveRoute struct {
	ModuleID   string                 `json:"moduleId"`
	Surface    string                 `json:"surface"`
	Prefix     string                 `json:"prefix"`
	Service    string                 `json:"service"`
	Origin     string                 `json:"origin"`
	Operations []ActiveRouteOperation `json:"operations"`
}

// ActiveRouteOperation is the minimum Registry metadata the Gateway needs to
// make an operation-level authentication decision. Keeping method, path and
// authentication together prevents a public read from widening its whole
// module prefix into an anonymous route.
type ActiveRouteOperation struct {
	Method         string `json:"method"`
	Path           string `json:"path"`
	Authentication string `json:"authentication"`
}

func Decode(data []byte) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Manifest{}, errors.New("module manifest must contain exactly one JSON document")
		}
		return Manifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (m Manifest) Validate() error {
	if m.APIVersion != APIVersion || m.Kind != "ModuleRelease" {
		return errors.New("unsupported module manifest version or kind")
	}
	if !moduleIDPattern.MatchString(m.Metadata.ID) || m.Metadata.Name == "" || !semverPattern.MatchString(m.Metadata.Version) {
		return errors.New("module id, name and version are required")
	}
	if m.Spec.Backend.Service == "" || m.Spec.Backend.Origin == "" {
		return errors.New("backend service and origin are required")
	}
	origin, err := url.Parse(m.Spec.Backend.Origin)
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || (origin.Path != "" && origin.Path != "/") {
		return errors.New("backend origin must be an http(s) origin without path, credentials, query or fragment")
	}
	seenRoutes := map[string]bool{}
	seenHTTP := map[string]bool{}
	for _, route := range m.Spec.Backend.HTTPRoutes {
		if !validSurfaces[route.Surface] || !strings.HasPrefix(route.Prefix, "/") {
			return fmt.Errorf("invalid route %s:%s", route.Surface, route.Prefix)
		}
		key := route.Surface + ":" + strings.TrimRight(route.Prefix, "/")
		if seenRoutes[key] {
			return fmt.Errorf("duplicate route %s", key)
		}
		seenRoutes[key] = true
		if len(route.Operations) == 0 {
			return fmt.Errorf("route %s must declare its operations", key)
		}
		for _, operation := range route.Operations {
			if operation.ID == "" {
				return fmt.Errorf("invalid or duplicate HTTP operation %q", operation.ID)
			}
			// Collection and member delete share <domain>.<feature>.delete (docs/24 §4.2).
			// Uniqueness is method+path, not operation id.
			httpKey := operation.Method + " " + operation.Path
			if seenHTTP[httpKey] {
				return fmt.Errorf("duplicate HTTP route %s %s", operation.Method, operation.Path)
			}
			seenHTTP[httpKey] = true
			if !validMethods[operation.Method] || !pathWithinPrefix(operation.Path, route.Prefix) {
				return fmt.Errorf("HTTP operation %s is outside route %s or uses an invalid method", operation.ID, key)
			}
			if operation.Summary == "" || operation.Description == "" || !validAuthentication[operation.Authentication] || !validIdempotency[operation.Idempotency] {
				return fmt.Errorf("HTTP operation %s has incomplete invocation metadata", operation.ID)
			}
			if operation.Authentication != "public" && operation.Authentication != "guest-session" && len(operation.RequiredPermissions) == 0 {
				return fmt.Errorf("HTTP operation %s must declare required permissions", operation.ID)
			}
			if err := validateCapabilityFields(operation.RequestFields, true); err != nil {
				return fmt.Errorf("HTTP operation %s request: %w", operation.ID, err)
			}
			if len(operation.Responses) == 0 {
				return fmt.Errorf("HTTP operation %s must declare responses", operation.ID)
			}
			if err := validateOperationNotifications(m.Metadata.ID, operation.ID, operation.Notifications); err != nil {
				return err
			}
			seenStatuses := map[int]bool{}
			for _, response := range operation.Responses {
				if response.Status < 100 || response.Status > 599 || response.Description == "" || seenStatuses[response.Status] {
					return fmt.Errorf("HTTP operation %s has an invalid response", operation.ID)
				}
				seenStatuses[response.Status] = true
				if err := validateCapabilityFields(response.Fields, false); err != nil {
					return fmt.Errorf("HTTP operation %s response: %w", operation.ID, err)
				}
			}
		}
	}
	if err := validateManifestNotificationUniqueness(m); err != nil {
		return err
	}
	if len(m.Spec.Permissions) == 0 {
		return errors.New("module must declare at least one permission")
	}
	definedPermissions := map[string]bool{}
	for _, permission := range m.Spec.Permissions {
		if !permissionPattern.MatchString(permission.Code) || permission.Name == "" || !permissionPattern.MatchString(permission.Resource+"."+permission.Action) || permission.Code != permission.Resource+"."+permission.Action || !strings.HasPrefix(permission.Code, m.Metadata.ID+".") {
			return fmt.Errorf("invalid permission definition %q", permission.Code)
		}
		if definedPermissions[permission.Code] {
			return fmt.Errorf("duplicate permission definition %q", permission.Code)
		}
		definedPermissions[permission.Code] = true
	}
	for _, route := range m.Spec.Backend.HTTPRoutes {
		for _, operation := range route.Operations {
			if (operation.Authentication == "public" || operation.Authentication == "guest-session") && len(operation.RequiredPermissions) == 0 {
				continue
			}
			if err := validatePermissionReferences(operation.RequiredPermissions, definedPermissions); err != nil {
				return fmt.Errorf("HTTP operation %s: %w", operation.ID, err)
			}
		}
	}
	if grpc := m.Spec.Backend.GRPC; grpc != nil {
		if grpc.Service == "" || !semverPattern.MatchString(grpc.ContractVersion) || grpc.Endpoint == "" || grpc.TransportSecurity == "" || len(grpc.Methods) == 0 {
			return errors.New("gRPC service must declare service, contract version, endpoint, transport security and methods")
		}
		seenMethods := map[string]bool{}
		for _, method := range grpc.Methods {
			expectedFullMethod := "/" + grpc.Service + "/" + method.Name
			if method.Name == "" || method.FullMethod != expectedFullMethod || seenMethods[method.FullMethod] || method.Summary == "" || method.Description == "" {
				return fmt.Errorf("invalid or duplicate gRPC method %q", method.FullMethod)
			}
			seenMethods[method.FullMethod] = true
			if !validGRPCInvocations[method.Invocation] || !validIdempotency[method.Idempotency] || method.RecommendedDeadlineMS <= 0 {
				return fmt.Errorf("gRPC method %s has incomplete invocation metadata", method.FullMethod)
			}
			if err := validatePermissionReferences(method.RequiredPermissions, definedPermissions); err != nil {
				return fmt.Errorf("gRPC method %s: %w", method.FullMethod, err)
			}
			if err := validateCapabilityFields(method.RequestFields, false); err != nil {
				return fmt.Errorf("gRPC method %s request: %w", method.FullMethod, err)
			}
			if err := validateCapabilityFields(method.ResponseFields, false); err != nil {
				return fmt.Errorf("gRPC method %s response: %w", method.FullMethod, err)
			}
		}
	}
	seenContributions := map[string]bool{}
	for _, contribution := range m.Spec.Contributions {
		if contribution.ID == "" || seenContributions[contribution.ID] {
			return fmt.Errorf("invalid or duplicate contribution %q", contribution.ID)
		}
		seenContributions[contribution.ID] = true
		if !validSurfaces[contribution.Surface] || contribution.Surface == "internal" || !validKinds[contribution.Kind] {
			return fmt.Errorf("invalid contribution %s", contribution.ID)
		}
		if contribution.Kind == "page" && !strings.HasPrefix(contribution.Route, "/") {
			return fmt.Errorf("page %s must declare an absolute route", contribution.ID)
		}
		if contribution.Navigation != nil {
			if contribution.Kind != "page" || !moduleIDPattern.MatchString(contribution.Navigation.GroupID) || contribution.Navigation.GroupTitle == "" {
				return fmt.Errorf("contribution %s has invalid navigation metadata", contribution.ID)
			}
		}
		if contribution.Kind != "page" && contribution.Outlet == "" {
			return fmt.Errorf("contribution %s must declare an outlet", contribution.ID)
		}
		if contribution.Title == "" || contribution.Description == "" || contribution.Frontend.Component == "" {
			return fmt.Errorf("contribution %s must describe its frontend component", contribution.ID)
		}
		if err := validateCapabilityFields(contribution.Frontend.Props, false); err != nil {
			return fmt.Errorf("contribution %s props: %w", contribution.ID, err)
		}
		seenEvents := map[string]bool{}
		for _, event := range contribution.Frontend.Events {
			if event.Name == "" || event.Description == "" || seenEvents[event.Name] {
				return fmt.Errorf("contribution %s has an invalid frontend event", contribution.ID)
			}
			seenEvents[event.Name] = true
			if err := validateCapabilityFields(event.Payload, false); err != nil {
				return fmt.Errorf("contribution %s event %s: %w", contribution.ID, event.Name, err)
			}
		}
		seenActions := map[string]bool{}
		for _, action := range contribution.Frontend.Actions {
			if action.ID == "" || action.Label == "" || action.Description == "" || action.Target == "" || seenActions[action.ID] || !validFrontendInvocations[action.Invocation] {
				return fmt.Errorf("contribution %s has an invalid frontend action %q", contribution.ID, action.ID)
			}
			seenActions[action.ID] = true
			if err := validatePermissionReferences(action.RequiredPermissions, definedPermissions); err != nil {
				return fmt.Errorf("contribution %s action %s: %w", contribution.ID, action.ID, err)
			}
			if err := validateCapabilityFields(action.Parameters, false); err != nil {
				return fmt.Errorf("contribution %s action %s: %w", contribution.ID, action.ID, err)
			}
		}
		if !validArtifactTypes[contribution.Artifact.Type] || contribution.Artifact.Name == "" || contribution.Artifact.Entry == "" || contribution.Artifact.Integrity == "" {
			return fmt.Errorf("invalid artifact for %s", contribution.ID)
		}
		entry, err := url.Parse(contribution.Artifact.Entry)
		if err != nil || (entry.Scheme != "http" && entry.Scheme != "https") || entry.Host == "" || entry.User != nil || !digestPattern.MatchString(contribution.Artifact.Integrity) {
			return fmt.Errorf("artifact %s must use an http(s) URL and a sha256 digest", contribution.ID)
		}
		if contribution.Artifact.Version != m.Metadata.Version {
			return fmt.Errorf("artifact %s version must match module release", contribution.ID)
		}
		if contribution.Artifact.Type == "remote-esm" && contribution.Artifact.ExportName == "" {
			return fmt.Errorf("remote-esm %s requires exportName", contribution.ID)
		}
		for _, permission := range contribution.RequiredPermissions {
			if !definedPermissions[permission] {
				return fmt.Errorf("contribution %s references undefined permission %q", contribution.ID, permission)
			}
		}
		for _, allowed := range contribution.AllowedRoutes {
			if len(allowed.Methods) == 0 || !strings.HasPrefix(allowed.Prefix, "/") {
				return fmt.Errorf("contribution %s has invalid allowed route", contribution.ID)
			}
			for _, method := range allowed.Methods {
				if !validMethods[strings.ToUpper(method)] {
					return fmt.Errorf("contribution %s has invalid method %q", contribution.ID, method)
				}
			}
			for _, permission := range allowed.RequiredPermissions {
				if !definedPermissions[permission] {
					return fmt.Errorf("contribution %s route references undefined permission %q", contribution.ID, permission)
				}
			}
			if !routeBelongsToSurface(m.Spec.Backend.HTTPRoutes, contribution.Surface, allowed.Prefix) {
				return fmt.Errorf("contribution %s allowed route is outside its registered surface", contribution.ID)
			}
		}
	}
	return nil
}

func validateOperationNotifications(moduleID, operationID string, notifications []NotificationDeclaration) error {
	seen := map[string]bool{}
	for _, item := range notifications {
		if !eventKeyPattern.MatchString(item.EventKey) || !strings.HasPrefix(item.EventKey, moduleID+".") {
			return fmt.Errorf("HTTP operation %s has invalid notification eventKey %q", operationID, item.EventKey)
		}
		if seen[item.EventKey] {
			return fmt.Errorf("HTTP operation %s declares duplicate notification %q", operationID, item.EventKey)
		}
		seen[item.EventKey] = true
		if strings.TrimSpace(item.Title) == "" {
			return fmt.Errorf("HTTP operation %s notification %s is missing a title", operationID, item.EventKey)
		}
		if !validNotifyDispatch[item.DefaultDispatch] {
			return fmt.Errorf("HTTP operation %s notification %s has invalid defaultDispatch", operationID, item.EventKey)
		}
		if len(item.AllowedChannels) == 0 {
			return fmt.Errorf("HTTP operation %s notification %s must declare allowedChannels", operationID, item.EventKey)
		}
		seenChannel := map[string]bool{}
		for _, channel := range item.AllowedChannels {
			if !validNotifyChannels[channel] || seenChannel[channel] {
				return fmt.Errorf("HTTP operation %s notification %s has invalid allowedChannels", operationID, item.EventKey)
			}
			seenChannel[channel] = true
		}
		seenVariable := map[string]bool{}
		for _, variable := range item.Variables {
			if !notifyVariablePattern.MatchString(variable) || seenVariable[variable] {
				return fmt.Errorf("HTTP operation %s notification %s has invalid variables", operationID, item.EventKey)
			}
			seenVariable[variable] = true
		}
	}
	return nil
}

func validateManifestNotificationUniqueness(m Manifest) error {
	seen := map[string]string{}
	for _, route := range m.Spec.Backend.HTTPRoutes {
		for _, operation := range route.Operations {
			for _, item := range operation.Notifications {
				if previous, exists := seen[item.EventKey]; exists {
					return fmt.Errorf("duplicate notification eventKey %q on %s and %s", item.EventKey, previous, operation.ID)
				}
				seen[item.EventKey] = operation.ID
			}
		}
	}
	return nil
}

func pathWithinPrefix(path, prefix string) bool {
	prefix = strings.TrimRight(prefix, "/")
	return strings.HasPrefix(path, "/") && (path == prefix || strings.HasPrefix(path, prefix+"/"))
}

func validateCapabilityFields(fields []CapabilityField, allowLocation bool) error {
	seen := map[string]bool{}
	for _, field := range fields {
		if field.Name == "" || field.Description == "" || !validFieldTypes[field.Type] || seen[field.Name] {
			return fmt.Errorf("invalid or duplicate capability field %q", field.Name)
		}
		seen[field.Name] = true
		if allowLocation {
			if !validFieldLocations[field.Location] {
				return fmt.Errorf("field %s has invalid location", field.Name)
			}
		} else if field.Location != "" {
			return fmt.Errorf("field %s must not declare a transport location", field.Name)
		}
	}
	return nil
}

func validatePermissionReferences(required []string, defined map[string]bool) error {
	if len(required) == 0 {
		return errors.New("required permissions are missing")
	}
	seen := map[string]bool{}
	for _, permission := range required {
		if !defined[permission] || seen[permission] {
			return fmt.Errorf("undefined or duplicate permission %q", permission)
		}
		seen[permission] = true
	}
	return nil
}

func routeBelongsToSurface(routes []HTTPRoute, surface, path string) bool {
	for _, route := range routes {
		prefix := strings.TrimRight(route.Prefix, "/")
		if route.Surface == surface && (path == prefix || strings.HasPrefix(path, prefix+"/")) {
			return true
		}
	}
	return false
}

func (m Manifest) Digest() (string, error) {
	copy := m
	sort.Slice(copy.Spec.Backend.HTTPRoutes, func(i, j int) bool {
		left, right := copy.Spec.Backend.HTTPRoutes[i], copy.Spec.Backend.HTTPRoutes[j]
		return left.Surface+left.Prefix < right.Surface+right.Prefix
	})
	sort.Slice(copy.Spec.Permissions, func(i, j int) bool { return copy.Spec.Permissions[i].Code < copy.Spec.Permissions[j].Code })
	sort.Slice(copy.Spec.Contributions, func(i, j int) bool { return copy.Spec.Contributions[i].ID < copy.Spec.Contributions[j].ID })
	data, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
