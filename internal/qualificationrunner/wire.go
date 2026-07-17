package qualificationrunner

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

func DecodeExecutionRequestJSON(data []byte) (ExecutionRequest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var request ExecutionRequest
	if err := decoder.Decode(&request); err != nil {
		return ExecutionRequest{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ExecutionRequest{}, errors.New("qualification execution request has trailing JSON data")
	}
	return request, nil
}

func EncodeReadinessArtifact(artifact ReadinessArtifact) ([]byte, error) {
	want, err := readinessHash(artifact)
	if err != nil {
		return nil, err
	}
	if artifact.ArtifactSHA256 != want {
		return nil, errors.New("readiness artifact hash mismatch")
	}
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func EncodeResultArtifact(artifact ResultArtifact) ([]byte, error) {
	want, err := resultHash(artifact)
	if err != nil {
		return nil, err
	}
	if artifact.ResultSHA256 != want {
		return nil, errors.New("result artifact hash mismatch")
	}
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
