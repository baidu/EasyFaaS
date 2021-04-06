/*
 * Copyright (c) 2020 Baidu, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logs

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	program = filepath.Base(os.Args[0])

	logging loggingT

	noplogger     *zap.Logger
	utlogger      *zap.Logger
	zaplogger     *zap.Logger
	summarylogger *zap.Logger
)

func init() {
	initCompatibleWithGlog()

	flag.StringVar(&logging.level, "log-level", "info",
		`log level ("debug", "info", "warn", "error", "panic", "panic", and "fatal").`)

	flag.BoolVar(&logging.enableTimetrack, "enable-timetrack", false,
		"whether enable timetrack log, may affect performance if set to 'true' (default false)")

	flag.BoolVar(&logging.readableLog, "enable-readable-log", false, ""+
		"Print human readable log, default false")

	// build default logger to prevent nil pointer
	// actually it's print to nowhere
	noplogger = zap.NewNop()

	// Default to ut log in unittest envrionment
	if strings.HasSuffix(program, ".test") {
		utlogger = zap.New(newUtCore()).WithOptions(zap.AddCallerSkip(1), zap.AddCaller())
		zaplogger = utlogger
		summarylogger = utlogger
	} else {
		zaplogger = noplogger
		summarylogger = noplogger
	}
}

// Compatible With Glog
func initCompatibleWithGlog() {
	flag.StringVar(&logging.logDir, "log_dir", "", "If non-empty, write log files in this directory")
	flag.BoolVar(&logging.toStderr, "logtostderr", false, "log to standard error instead of files")
	flag.BoolVar(&logging.alsoToStderr, "alsologtostderr", false, "log to standard error as well as files")
	flag.Var(&logging.verbosity, "v", "log level for V logs")
}

// InitLogs need to be called explicit in the main application
func InitLogs() {
	if zaplogger != noplogger {
		return
	}
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if err := level.UnmarshalText([]byte(logging.level)); err != nil {
		fmt.Printf("log level '%s' invalid", logging.level)
	}

	var core zapcore.Core
	if logging.toStderr || logging.alsoToStderr {
		core = newStderrCore(level)
	}
	if !logging.toStderr && len(logging.logDir) > 0 {
		// creating log path
		if err := os.MkdirAll(logging.logDir, os.ModePerm); err != nil {
			fmt.Printf("create dir failed, err=%s\n", err.Error())
		} else {
			if core == nil {
				core = newFileCore(level)
			} else {
				core = zapcore.NewTee(core, newFileCore(level))
			}
		}
	}
	zaplogger = zap.New(core).WithOptions(zap.AddCallerSkip(1), zap.AddCaller())
}

// InitSummaryLogs init a logger to record summary
func InitSummaryLogs() {
	if summarylogger != noplogger {
		return
	}
	if len(logging.logDir) == 0 {
		return
	}
	// creating log path
	if err := os.MkdirAll(logging.logDir, os.ModePerm); err != nil {
		fmt.Printf("create dir failed, err=%s\n", err.Error())
		return
	}
	summarylogger = zap.New(newSummaryFileCore()).WithOptions(zap.AddCallerSkip(1), zap.AddCaller())
}

func newStderrCore(level zap.AtomicLevel) zapcore.Core {
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	if logging.readableLog {
		encoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}
	writer := zapcore.AddSync(os.Stderr)
	return zapcore.NewCore(encoder, writer, level.Level())
}

func newUtCore() zapcore.Core {
	var (
		logdir   = "/tmp/kun-unit-test/"
		filename string
		newest   time.Time
	)
	files, _ := ioutil.ReadDir(logdir)
	for _, f := range files {
		fs, err := os.Stat(logdir + f.Name())
		if err != nil {
			panic(err)
		}
		ft := fs.ModTime()
		if ft.After(newest) {
			filename = logdir + f.Name()
			newest = ft
		}
	}
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	if len(filename) == 0 {
		return nil
	}

	file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		panic(filename + " cannot be opened for write")
	}
	writer := zapcore.AddSync(file)
	return zapcore.NewCore(encoder, writer, zap.DebugLevel)
}

func newFileCore(level zap.AtomicLevel) zapcore.Core {
	errenabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel
	})
	infoenabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= level.Level()
	})

	errfilewriter := zapcore.AddSync(
		NewDiodeWriter(NewRotate(logging.logDir, program, "ERROR"), 3000, 5*time.Millisecond, func(missed int) {
			fmt.Printf("Logger Dropped %d messages", missed)
		}),
	)
	infofilewriter := zapcore.AddSync(
		NewDiodeWriter(NewRotate(logging.logDir, program, "INFO"), 3000, 5*time.Millisecond, func(missed int) {
			fmt.Printf("Logger Dropped %d messages", missed)
		}),
	)

	jsonEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	if logging.readableLog {
		jsonEncoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}
	return zapcore.NewTee(
		zapcore.NewCore(jsonEncoder, infofilewriter, infoenabler),
		zapcore.NewCore(jsonEncoder, errfilewriter, errenabler),
	)
}

func newSummaryFileCore() zapcore.Core {
	enabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.InfoLevel
	})

	writer := zapcore.AddSync(
		NewDiodeWriter(NewRotate(logging.logDir, program, "SUMMARY"), 3000, 5*time.Millisecond, func(missed int) {
			fmt.Printf("Logger Dropped %d messages", missed)
		}),
	)

	jsonEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	if logging.readableLog {
		jsonEncoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	}
	return zapcore.NewCore(jsonEncoder, writer, enabler)
}

// loggingT collects all the global state of the logging setup.
type loggingT struct {
	// Boolean flags. Not handled atomically because the flag.Value interface
	// does not let us avoid the =true, and that shorthand is necessary for
	// compatibility. TODO: does this matter enough to fix? Seems unlikely.
	toStderr     bool // The -logtostderr flag.
	alsoToStderr bool // The -alsologtostderr flag.

	// If non-empty, write log files in this directory
	logDir string

	// mu protects the remaining elements of this structure and is
	// used to synchronize logging.
	mu sync.Mutex

	// These flags are modified only under lock, although verbosity may be fetched
	// safely using atomic.LoadInt32.
	verbosity Level // V logging level, the value of the -v flag/

	// zap log level, valid options are
	// ("debug", "info", "warn", "error", "dpanic", "panic", and "fatal").
	level string

	// whether enable timetrack log, may affect performace if set to 'true'
	enableTimetrack bool

	// whether enable human readable log
	readableLog bool
}

type LoggerConfig struct {
	LogDir    string
	Verbosity Level
}

func ConfigDefaultLogger(lc *LoggerConfig) {
	logging.logDir = lc.LogDir
	logging.verbosity = lc.Verbosity
}

// NewSummaryLogger creates a new log.Logger
func NewSummaryLogger() *Logger {
	return (*Logger)(summarylogger)
}

// NewLogger creates a new log.Logger
func NewLogger() *Logger {
	return (*Logger)(zaplogger)
}

// Level is exported because it appears in the arguments to V and is
// the type of the v flag, which can be set programmatically.
// It's a distinct type because we want to discriminate it from logType.
// Variables of type level are only changed under logging.mu.
// The -v flag is read only with atomic ops, so the state of the logging
// module is consistent.

// Level is treated as a sync/atomic int32.

// Level specifies a level of verbosity for V logs. *Level implements
// flag.Value; the -v flag is of type Level and should be modified
// only through the flag.Value interface.
type Level int32

// get returns the value of the Level.
func (l *Level) get() Level {
	return Level(atomic.LoadInt32((*int32)(l)))
}

// set sets the value of the Level.
func (l *Level) set(val Level) {
	atomic.StoreInt32((*int32)(l), int32(val))
}

// String is part of the flag.Value interface.
func (l *Level) String() string {
	return strconv.FormatInt(int64(*l), 10)
}

// Get is part of the flag.Value interface.
func (l *Level) Get() interface{} {
	return *l
}

// Set is part of the flag.Value interface.
func (l *Level) Set(value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	logging.mu.Lock()
	defer logging.mu.Unlock()
	logging.verbosity.set(Level(v))
	return nil
}

// Logger xxx
type Logger zap.Logger

// V is a replacement of glog.V()
func V(level Level) *Logger {
	// return Verbose{level
	if logging.verbosity.get() >= level {
		return (*Logger)(zaplogger)
	}
	return nil
}

// V is a replacement of glog.V()
func (l *Logger) V(level Level) *Logger {
	if logging.verbosity.get() >= level {
		return l
	}
	return nil
}

// Check if need to print log
func Check(v *Logger) bool {
	return v != nil
}

// Infof is equivalent to the global Infof function, guarded by the value of v.
// See the documentation of V for usage.
func (l *Logger) Infof(template string, args ...interface{}) {
	if Check(l) {
		(*zap.Logger)(l).Sugar().Infof(template, args...)
	}
}

// Info is equivalent to the global Info function, guarded by the value of v.
// See the documentation of V for usage.
func (l *Logger) Info(msg string, fields ...zap.Field) {
	if Check(l) {
		(*zap.Logger)(l).Info(msg, fields...)
	}
}

// Debugf is equivalent to the global Debugf function, guarded by the value of v.
// See the documentation of V for usage.
func (l *Logger) Debugf(template string, args ...interface{}) {
	if Check(l) {
		(*zap.Logger)(l).Sugar().Debugf(template, args...)
	}
}

// Debug is equivalent to the global Debug function, guarded by the value of v.
// See the documentation of V for usage.
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	if Check(l) {
		(*zap.Logger)(l).Debug(msg, fields...)
	}
}

// Warn is equivalent to the global Warn function, guarded by the value of v.
// See the documentation of V for usage.
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	if Check(l) {
		(*zap.Logger)(l).Warn(msg, fields...)
	}
}

func (l *Logger) Warnf(template string, args ...interface{}) {
	if Check(l) {
		(*zap.Logger)(l).Sugar().Warnf(template, args...)
	}
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	if Check(l) {
		(*zap.Logger)(l).Error(msg, fields...)
	}
}

func (l *Logger) Errorf(template string, args ...interface{}) {
	if Check(l) {
		(*zap.Logger)(l).Sugar().Errorf(template, args...)
	}
}

func (l *Logger) Fatalf(template string, args ...interface{}) {
	if Check(l) {
		(*zap.Logger)(l).Sugar().Fatalf(template, args...)
	}
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	if Check(l) {
		return (*Logger)((*zap.Logger)(l).With(fields...))
	}
	return nil
}

func (l *Logger) WithField(k, v string) *Logger {
	if Check(l) {
		return (*Logger)((*zap.Logger)(l).With(zap.String(k, v)))
	}
	return nil
}

func (l *Logger) WithStringField(k, v string) *Logger {
	if Check(l) {
		return (*Logger)((*zap.Logger)(l).With(zap.String(k, v)))
	}
	return nil
}

func (l *Logger) TimeTrack(start time.Time, thumbnail string, fields ...zap.Field) {
	if logging.enableTimetrack {
		endTime := time.Now().UTC()
		endTimestamp := endTime.UnixNano() / 1e6
		startTimestamp := start.UTC().UnixNano() / 1e6
		sub := endTime.Sub(start).Seconds() * 1e3

		fields = append(fields,
			zap.String("timeTrack", thumbnail),
			zap.Int64("startT", startTimestamp),
			zap.Int64("endT", endTimestamp),
			zap.Int("sub", int(sub)),
		)
		(*zap.Logger)(l).Info(thumbnail, fields...)
	}
}

func (l *Logger) AddCallerSkip(n int) *Logger {
	if Check(l) {
		return (*Logger)((*zap.Logger)(l).WithOptions(zap.AddCallerSkip(n)))
	}
	return nil
}

func FlushLogs() {
	zaplogger.Sync()
	summarylogger.Sync()
}

func Info(msg string, fields ...zap.Field) {
	zaplogger.Info(msg, fields...)
}

// Infof is a repalcement of glog.Infof()
func Infof(template string, args ...interface{}) {
	zaplogger.Sugar().Infof(template, args...)
}

func Warn(msg string, fields ...zap.Field) {
	zaplogger.Warn(msg, fields...)
}

func Warnf(template string, args ...interface{}) {
	zaplogger.Sugar().Warnf(template, args...)
}

func Error(msg string, fields ...zap.Field) {
	zaplogger.Error(msg, fields...)
}

func Errorf(template string, args ...interface{}) {
	zaplogger.Sugar().Errorf(template, args...)
}

func Debug(msg string, fields ...zap.Field) {
	zaplogger.Debug(msg, fields...)
}

func Debugf(template string, args ...interface{}) {
	zaplogger.Sugar().Debugf(template, args...)
}

func Fatal(msg string, fields ...zap.Field) {
	zaplogger.Fatal(msg, fields...)
}

func Fatalf(template string, args ...interface{}) {
	zaplogger.Sugar().Fatalf(template, args...)
}

func Exitf(template string, args ...interface{}) {
	zaplogger.Sugar().Fatalf(template, args...)
}

// TimeTrack
func TimeTrack(start time.Time, name string) {
	if logging.enableTimetrack {
		elapsed := time.Since(start)
		zaplogger.Sugar().Infof("%s took %fms", name, elapsed.Seconds()*1e3)
	}
}
