package cbinsights

import (
	"time"
)

type jsonQueryMetrics struct {
	ElapsedTime      string `json:"elapsedTime"`
	ExecutionTime    string `json:"executionTime"`
	ResultCount      uint64 `json:"resultCount"`
	ResultSize       uint64 `json:"resultSize"`
	MutationCount    uint64 `json:"mutationCount,omitempty"`
	SortCount        uint64 `json:"sortCount,omitempty"`
	ErrorCount       uint64 `json:"errorCount,omitempty"`
	WarningCount     uint64 `json:"warningCount,omitempty"`
	ProcessedObjects uint64 `json:"processedObjects,omitempty"`
}

type jsonQueryWarning struct {
	Code    uint32 `json:"code"`
	Message string `json:"msg"`
}

type jsonQueryResponse struct {
	RequestID       string             `json:"requestID"`
	ClientContextID string             `json:"clientContextID"`
	Status          string             `json:"status"`
	Warnings        []jsonQueryWarning `json:"warnings"`
	Metrics         jsonQueryMetrics   `json:"metrics"`
	Signature       interface{}        `json:"signature"`
	Handle          string             `json:"handle,omitempty"`
}

func (meta *QueryMetadata) fromData(data jsonQueryResponse) {
	metrics := QueryMetrics{
		ElapsedTime:      0,
		ExecutionTime:    0,
		ResultCount:      0,
		ResultSize:       0,
		ProcessedObjects: 0,
	}
	metrics.fromData(data.Metrics)

	warnings := make([]QueryWarning, len(data.Warnings))
	for wIdx, jsonWarning := range data.Warnings {
		warnings[wIdx].fromData(jsonWarning)
	}

	meta.RequestID = data.RequestID
	meta.Metrics = metrics
	meta.Warnings = warnings
}

func (metrics *QueryMetrics) fromData(data jsonQueryMetrics) {
	elapsedTime, _ := time.ParseDuration(data.ElapsedTime)
	executionTime, _ := time.ParseDuration(data.ExecutionTime)
	metrics.ElapsedTime = elapsedTime
	metrics.ExecutionTime = executionTime
	metrics.ResultCount = data.ResultCount
	metrics.ResultSize = data.ResultSize
	metrics.ProcessedObjects = data.ProcessedObjects
}

func (warning *QueryWarning) fromData(data jsonQueryWarning) {
	warning.Code = data.Code
	warning.Message = data.Message
}
