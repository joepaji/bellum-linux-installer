package core

// ErrorLevel defines the severity level for error handling
type ErrorLevel int

const (
	// ErrorLevelCritical indicates a critical error that should halt execution
	ErrorLevelCritical ErrorLevel = 1
	// ErrorLevelWarning indicates a non-critical error that can be ignored or logged
	ErrorLevelWarning ErrorLevel = 2
	// ErrorLevelInfo indicates an informational message
	ErrorLevelInfo ErrorLevel = 3
)

// handleError handles errors based on their severity level.
// Critical errors are logged and returned.
// Warning errors are logged but execution continues.
// Info-level messages are logged for tracing.
func handleError(err error, level ErrorLevel, logger *Logger) {
	if err == nil {
		return
	}

	if logger == nil {
		return
	}

	switch level {
	case ErrorLevelCritical:
		logger.Error(err.Error())
	case ErrorLevelWarning:
		logger.Warn(err.Error())
	case ErrorLevelInfo:
		logger.Info(err.Error())
	}
}

// HandleErrorResult is the result of handling an error with a specific level.
type HandleErrorResult struct {
	ShouldReturn bool  // true if the error should cause the function to return
	Error        error // The original error
}

// HandleError handles an error and returns whether execution should continue.
// Critical errors return true (should return), Warning errors return false (continue).
func HandleError(err error, level ErrorLevel, logger *Logger) HandleErrorResult {
	handleError(err, level, logger)
	return HandleErrorResult{
		ShouldReturn: level == ErrorLevelCritical,
		Error:        err,
	}
}

// LogAndReturn logs the error at the given level and returns it.
// Use this to replace the pattern: logger.Error(msg); return err
// For warnings: LogAndReturn(err, ErrorLevelWarning, logger)
func LogAndReturn(err error, level ErrorLevel, logger *Logger) error {
	if err == nil {
		return nil
	}
	handleError(err, level, logger)
	return err
}
