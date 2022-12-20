package builder

import (
	"fmt"
	"net/url"
	"strings"

	cev2event "github.com/cloudevents/sdk-go/v2/event"

	"github.com/kyma-project/kyma/components/event-publisher-proxy/pkg/application"
	"github.com/kyma-project/kyma/components/eventing-controller/logger"
	"github.com/kyma-project/kyma/components/eventing-controller/pkg/backend/cleaner"
	"go.uber.org/zap"
)

// Perform a compile-time check.
var _ CloudEventBuilder = &GenericBuilder{}

var (
	// jsBuilderName used as the logger name.
	genericBuilderName = "generic-type-builder"
)

func NewGenericBuilder(typePrefix string, cleaner cleaner.Cleaner, applicationLister *application.Lister, logger *logger.Logger) CloudEventBuilder {
	return &GenericBuilder{
		typePrefix:        typePrefix,
		applicationLister: applicationLister,
		logger:            logger,
		cleaner:           cleaner,
	}
}

func (gb *GenericBuilder) Build(event cev2event.Event) (*cev2event.Event, error) {
	// get unescaped strings from cloud event
	eventSource, err := url.QueryUnescape(event.Source())
	if err != nil {
		return nil, err
	}
	eventType, err := url.QueryUnescape(event.Type())
	if err != nil {
		return nil, err
	}

	// format logger
	namedLogger := gb.namedLogger(eventSource, eventType)

	// clean the source
	cleanSource, err := gb.cleaner.CleanSource(gb.GetAppNameOrSource(eventSource, namedLogger))
	if err != nil {
		return nil, err
	}

	// clean the event type
	cleanEventType, err := gb.cleaner.CleanEventType(eventType)
	if err != nil {
		return nil, err
	}

	// build event type
	finalEventType := gb.getFinalSubject(cleanSource, cleanEventType)

	// validate if the segments are not empty
	segments := strings.Split(finalEventType, ".")
	if CheckForEmptySegments(segments) {
		return nil, fmt.Errorf("event type cannot have empty segments after cleaning: %s", finalEventType)
	}
	namedLogger.Debugf("using event type ==: %s", finalEventType)

	ceEvent := event.Clone()
	ceEvent.SetType(finalEventType)
	ceEvent.SetSource(cleanSource)

	return &ceEvent, nil
}

// getFinalSubject return the final prefixed event type
func (gb *GenericBuilder) getFinalSubject(source, eventType string) string {
	return fmt.Sprintf("%s.%s.%s", gb.typePrefix, source, eventType)
}

// GetAppNameOrSource returns the application name if exists, otherwise returns source name
func (gb *GenericBuilder) GetAppNameOrSource(source string, namedLogger *zap.SugaredLogger) string {
	var appName = source
	if appObj, err := gb.applicationLister.Get(source); err == nil && appObj != nil {
		appName = application.GetCleanTypeOrName(appObj)
		namedLogger.With("application", source).Debug("Using application name: %s as source.", appName)
	} else {
		namedLogger.With("application", source).Debug("Cannot find application.")
	}
	return appName
}

func (gb *GenericBuilder) namedLogger(source, eventType string) *zap.SugaredLogger {
	return gb.logger.WithContext().Named(genericBuilderName).With("source", source, "type", eventType)
}
