package log

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/go-logr/zapr"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/klog/v2"
)

var (
	// Runtime logger for runtime logging
	Runtime *zap.Logger
)

var (
	runtimeWriter *lumberjack.Logger
	runtimeSyncer *MutableWriteSyncer

	// UseFileLogger indicate use custom file logger or not
	UseFileLogger                        bool
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
	klog.SetLogger(zapr.NewLogger(Runtime))
	return nil
}

// MutableWriteSyncer a WriteSyncer implementation support change inner WriteSyncer on the fly
type MutableWriteSyncer struct {
	syncer atomic.Value
}

func newMutableWriteSyncer(defaultSyncer zapcore.WriteSyncer) *MutableWriteSyncer {
	mws := &MutableWriteSyncer{}
	mws.syncer.Store(&defaultSyncer)
	return mws
}

func (mws *MutableWriteSyncer) get() zapcore.WriteSyncer {
	return *(mws.syncer.Load().(*zapcore.WriteSyncer))
}

func (mws *MutableWriteSyncer) setWriteSyncer(newSyncer zapcore.WriteSyncer) {
	mws.syncer.Store(&newSyncer)
}

// Write implement WriteSyncer.Write
func (mws *MutableWriteSyncer) Write(p []byte) (n int, err error) {
	return mws.get().Write(p)
}

// Sync implement WriteSyncer.Sync
func (mws *MutableWriteSyncer) Sync() error {
	return mws.get().Sync()
}

func init() {
	runtimeSyncer = newMutableWriteSyncer(zapcore.Lock(zapcore.AddSync(os.Stdout)))
	jsonEncoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})

	runtimeCore := zapcore.NewCore(
		jsonEncoder,
		runtimeSyncer,
		zapcore.InfoLevel,
	)
	Runtime = zap.New(runtimeCore, zap.AddCaller())
}

// Ref: https://github.com/uber-go/zap/blob/master/FAQ.md#does-zap-support-log-rotation
func initWithRotation(file string, runtimeRemaindays, runtimeMaxSize, runtimeMaxBackups int) (err error) {
	if file == "" {
		return fmt.Errorf("must specify logfile")
	}

	if file != "" {
		runtimeWriter = &lumberjack.Logger{
			Filename:   file,
			MaxSize:    runtimeMaxSize,    // megabytes
			MaxBackups: runtimeMaxBackups, // files number
			MaxAge:     runtimeRemaindays, // days
		}
		runtimeSyncer.setWriteSyncer(zapcore.AddSync(runtimeWriter))
	}

	return nil
}

// Final finalizer of this module
func Final() {
	if Runtime != nil {
		_ = Runtime.Sync()
	}

	if runtimeWriter != nil {
		_ = runtimeWriter.Close()
	}
}
