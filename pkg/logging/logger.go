package logging

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"sync"
)

const (
	Reset  = "\033[0m"
	Green  = "\033[32m" // INFO
	Yellow = "\033[33m" // WARN
	Red    = "\033[31m" // ERROR
)

type CustomTextFormatter struct{}

func (f *CustomTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var color string
	switch entry.Level {
	case logrus.InfoLevel:
		color = Green
	case logrus.WarnLevel:
		color = Yellow
	case logrus.ErrorLevel:
		color = Red
	default:
		color = Reset
	}

	logLine := fmt.Sprintf("%s%s%s %s %s",
		color, strings.ToUpper(entry.Level.String()), Reset,
		entry.Message,
		entry.Time.Format("2006-01-02 15:04:05"),
	)

	for _, value := range entry.Data {
		logLine += fmt.Sprintf(" %v", value)
	}

	return []byte(logLine + "\n"), nil
}

var (
	logInstance *logrus.Logger
	once        sync.Once
)

func GetLogger() *logrus.Logger {
	once.Do(func() {
		logInstance = logrus.New()
		logInstance.SetOutput(os.Stdout)
		logInstance.SetFormatter(&CustomTextFormatter{})
	})
	return logInstance
}

func Middleware(c *gin.Context) {
	logInstance.WithFields(logrus.Fields{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
	}).Info("Incoming request")

	c.Next()

	logInstance.WithFields(logrus.Fields{
		"status": c.Writer.Status(),
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
	}).Info("Request handled")
}
