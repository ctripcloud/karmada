package log

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/klog/v2"
)

var (
	// UseFileLogger indicate use custom file logger or not
	UseFileLogger bool

	runtimeSyncer                        zapcore.WriteSyncer
	globalFlags                          *pflag.FlagSet
	logRemain, logMaxSize, logMaxBackups int
)

// AddFlags add pkg flags
func AddFlags(fs *pflag.FlagSet) {
	flagSetShim := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSetShim.BoolVar(&UseFileLogger, "file-logger", false, "use customized file logger instead of klog default logger, log-dir must specified")
	flagSetShim.IntVar(&logRemain, "log-remain", 7, "Days to remain logs in LogDir.")
	flagSetShim.IntVar(&logMaxSize, "log-maxsize", 2048, "Maximum size in megabytes of the log file before rotated.")
	flagSetShim.IntVar(&logMaxBackups, "log-backups", 20, "Maximum number of old log files to retain.")

	fs.AddGoFlagSet(flagSetShim)
	globalFlags = fs
}

// InitLogger init custom file logger
func InitLogger() error {
	path := globalFlags.Lookup("log_dir").Value.String()
	if len(path) == 0 {
		return fmt.Errorf("log-dir not specified")
	}
	err := initWithRotation(filepath.Join(path, "runtime.log"), logRemain, logMaxSize, logMaxBackups)
	if err != nil {
		return err
	}
	klog.SetOutputBySeverity("INFO", runtimeSyncer)
	klog.SetOutputBySeverity("WARNING", io.Discard)
	klog.SetOutputBySeverity("ERROR", io.Discard)
	return nil
}

func initWithRotation(file string, runtimeRemaindays, runtimeMaxSize, runtimeMaxBackups int) (err error) {
	if file == "" {
		return fmt.Errorf("must specify logfile")
	}

	// Ref: https://github.com/uber-go/zap/blob/master/FAQ.md#does-zap-support-log-rotation
	runtimeWriter := &lumberjack.Logger{
		Filename:   file,
		MaxSize:    runtimeMaxSize,    // megabytes
		MaxBackups: runtimeMaxBackups, // files number
		MaxAge:     runtimeRemaindays, // days
	}
	runtimeSyncer = zapcore.AddSync(runtimeWriter)

	return nil
}

// Final finalizer of this module
func Final() {
	if runtimeSyncer != nil {
		_ = runtimeSyncer.Sync()
	}
}
