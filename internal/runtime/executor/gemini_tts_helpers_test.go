package executor

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestIsGeminiTTSPayload_DetectsByModelName(t *testing.T) {
	if !isGeminiTTSPayload("gemini-2.5-flash-preview-tts", nil) {
		t.Fatalf("expected model name to be detected as TTS")
	}
}

func TestIsGeminiTTSPayload_DetectsBySpeechConfig(t *testing.T) {
	payload := []byte(`{"generationConfig":{"speechConfig":{"voiceConfig":{"prebuiltVoiceConfig":{"voiceName":"Aoede"}}}}}`)
	if !isGeminiTTSPayload("gemini-2.5-flash", payload) {
		t.Fatalf("expected speechConfig to be detected as TTS")
	}
}

func TestIsGeminiTTSPayload_DetectsByAudioResponseModality(t *testing.T) {
	payload := []byte(`{"generationConfig":{"responseModalities":["AUDIO"]}}`)
	if !isGeminiTTSPayload("gemini-2.5-flash", payload) {
		t.Fatalf("expected AUDIO response modality to be detected as TTS")
	}
}

func TestStripGeminiTTSUnsupportedFields_RemovesUnsupportedFields(t *testing.T) {
	payload := []byte(`{"tools":[{"function_declarations":[]}],"toolConfig":{"functionCallingConfig":{"mode":"ANY"}},"safetySettings":[{"category":"HARM_CATEGORY_HATE_SPEECH","threshold":"OFF"}],"generationConfig":{"responseModalities":["AUDIO"],"speechConfig":{"voiceConfig":{"prebuiltVoiceConfig":{"voiceName":"Aoede"}}}}}`)
	out := stripGeminiTTSUnsupportedFields(payload)

	if gjson.GetBytes(out, "tools").Exists() {
		t.Fatalf("expected tools to be removed")
	}
	if gjson.GetBytes(out, "toolConfig").Exists() {
		t.Fatalf("expected toolConfig to be removed")
	}
	if gjson.GetBytes(out, "safetySettings").Exists() {
		t.Fatalf("expected safetySettings to be removed")
	}
	if !gjson.GetBytes(out, "generationConfig.speechConfig").Exists() {
		t.Fatalf("expected generationConfig.speechConfig to be preserved")
	}
}

