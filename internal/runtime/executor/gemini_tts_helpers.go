package executor

import (
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func isGeminiTTSPayload(model string, payload []byte) bool {
	lowerModel := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(lowerModel, "tts") {
		return true
	}

	if len(payload) == 0 {
		return false
	}

	if gjson.GetBytes(payload, "generationConfig.speechConfig").Exists() {
		return true
	}

	modalities := gjson.GetBytes(payload, "generationConfig.responseModalities")
	if !modalities.Exists() || !modalities.IsArray() {
		return false
	}
	for _, modality := range modalities.Array() {
		if strings.EqualFold(strings.TrimSpace(modality.String()), "AUDIO") {
			return true
		}
	}

	return false
}

func stripGeminiTTSUnsupportedFields(payload []byte) []byte {
	out := payload
	for _, path := range []string{"tools", "toolConfig", "safetySettings"} {
		updated, err := sjson.DeleteBytes(out, path)
		if err != nil {
			continue
		}
		out = updated
	}
	return out
}

