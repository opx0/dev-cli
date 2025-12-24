package infra

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp     time.Time               `json:"timestamp"`
	Container     string                  `json:"container"`
	Stream        string                  `json:"stream"`
	Message       string                  `json:"message"`
	GPUSnapshot   *GPUStats               `json:"gpu,omitempty"`
	ContainerSnap *ContainerStatsSnapshot `json:"container_stats,omitempty"`
}

type LogSink interface {
	Write(entry LogEntry) error
	Close() error
}

type FileLogSink struct {
	mu       sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	filePath string
}

func NewFileLogSink(path string) (*FileLogSink, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	return &FileLogSink{
		file:     file,
		writer:   bufio.NewWriter(file),
		filePath: path,
	}, nil
}

func (s *FileLogSink) Write(entry LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	line := fmt.Sprintf("[%s] [%s] %s\n",
		entry.Timestamp.Format(time.RFC3339),
		entry.Stream,
		entry.Message,
	)

	_, err := s.writer.WriteString(line)
	return err
}

func (s *FileLogSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.writer.Flush(); err != nil {
		return err
	}
	return s.file.Close()
}

func (s *FileLogSink) Path() string {
	return s.filePath
}

type JSONLLogSink struct {
	mu       sync.Mutex
	file     *os.File
	encoder  *json.Encoder
	filePath string
}

func NewJSONLLogSink(path string) (*JSONLLogSink, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	return &JSONLLogSink{
		file:     file,
		encoder:  json.NewEncoder(file),
		filePath: path,
	}, nil
}

func (s *JSONLLogSink) Write(entry LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.encoder.Encode(entry)
}

func (s *JSONLLogSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.file.Close()
}

func (s *JSONLLogSink) Path() string {
	return s.filePath
}

type MultiLogSink struct {
	mu    sync.Mutex
	sinks []LogSink
}

func NewMultiLogSink(sinks ...LogSink) *MultiLogSink {
	return &MultiLogSink{
		sinks: sinks,
	}
}

func (s *MultiLogSink) AddSink(sink LogSink) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sinks = append(s.sinks, sink)
}

func (s *MultiLogSink) Write(entry LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var firstErr error
	for _, sink := range s.sinks {
		if err := sink.Write(entry); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *MultiLogSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var firstErr error
	for _, sink := range s.sinks {
		if err := sink.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type WriterLogSink struct {
	mu     sync.Mutex
	writer io.Writer
	format string
}

func NewWriterLogSink(w io.Writer, format string) *WriterLogSink {
	if format == "" {
		format = "text"
	}
	return &WriterLogSink{
		writer: w,
		format: format,
	}
}

func (s *WriterLogSink) Write(entry LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var data []byte
	var err error

	if s.format == "jsonl" {
		data, err = json.Marshal(entry)
		if err != nil {
			return err
		}
		data = append(data, '\n')
	} else {
		line := fmt.Sprintf("[%s] [%s] %s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Stream,
			entry.Message,
		)
		data = []byte(line)
	}

	_, err = s.writer.Write(data)
	return err
}

func (s *WriterLogSink) Close() error {
	return nil
}

func CreateLogSink(config Config, containerName string) (LogSink, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("docker-%s-%s", containerName, timestamp)

	var path string
	if config.LogFormat == "jsonl" {
		path = filepath.Join(config.DevlogsDir, filename+".jsonl")
		return NewJSONLLogSink(path)
	}
	path = filepath.Join(config.DevlogsDir, filename+".log")
	return NewFileLogSink(path)
}
