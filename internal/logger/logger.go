/*
 * Copyright 2025 Comcast Cable Communications Management, LLC
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

package logger

import (
	"context"
	"fmt"
	"io"
	"os"

	"log/slog"
)

type Logger struct {
	*slog.Logger
	lvlVar *slog.LevelVar
}

func NewLogger() *Logger {
	lvlVar := new(slog.LevelVar)
	return &Logger{
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvlVar})),
		lvlVar: lvlVar,
	}
}

func NewLoggerWithIOWriter(w io.Writer) *Logger {
	lvlVar := new(slog.LevelVar)
	return &Logger{
		Logger: slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvlVar})),
		lvlVar: lvlVar,
	}
}

func (l *Logger) Fatal(v ...any) {
	l.LogAttrs(context.Background(), slog.LevelError, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...any) {
	l.LogAttrs(context.Background(), slog.LevelError, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *Logger) Level() slog.Level {
	return l.lvlVar.Level()
}

func (l *Logger) SetLevel(level slog.Level) {
	l.lvlVar.Set(level)
}

func (l *Logger) Slog() *slog.Logger {
	return l.Logger
}

func (l *Logger) With(args ...any) *Logger {
	if len(args) == 0 {
		return l
	}
	c := *l
	c.Logger = l.Logger.With(args...)
	return &c
}

func (l *Logger) WithGroup(name string) *Logger {
	if name == "" {
		return l
	}
	c := *l
	c.Logger = l.Logger.WithGroup(name)
	return &c
}
