package parsers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Chative-core-poc-v1/server/internal/agent/model"
	errx "github.com/Chative-core-poc-v1/server/internal/core/error"
	logx "github.com/Chative-core-poc-v1/server/pkg/logger"
)

const (
	recDelim = "##"
	tupDelim = "<||>"
	endDelim = "<|COMPLETE|>"
)

// basic safety limits to avoid pathological inputs
const (
	maxContentLen = 128 * 1024 // 128KB
	maxRecords    = 500        // maximum number of records to process
	maxTupleLen   = 8 * 1024   // 8KB per tuple
	maxMetaLen    = 4 * 1024   // 4KB metadata JSON
	maxErrSnippet = 200        // limit error snippet size
)

type rawTuple struct {
	Type  string
	Parts []string
}

func parseRawTuple(s string) (*rawTuple, error) {
	if s == "" {
		return nil, fmt.Errorf("empty tuple")
	}
	// enforce a sane upper bound per record
	if len(s) > maxTupleLen {
		return nil, fmt.Errorf("tuple too large")
	}

	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '(' || s[len(s)-1] != ')' {
		return nil, fmt.Errorf("invalid tuple parens")
	}
	// remove the outermost parens only
	inner := s[1 : len(s)-1]
	// limit splitting to at most 5 segments so metadata can contain delimiters
	parts := strings.SplitN(inner, tupDelim, 5)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid tuple parts")
	}
	return &rawTuple{Type: strings.TrimSpace(parts[0]), Parts: parts}, nil
}

func mustValidUTF8(s string, name string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("%s invalid utf8", name)
	}
	return nil
}

func parseFloat(s string, name string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("%s parse: %w", name, err)
	}
	return v, nil
}

func parseFloatInRange(s, name string, min, max float64) (float64, error) {
	v, err := parseFloat(s, name)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("%s invalid number", name)
	}
	if v < min || v > max {
		return 0, fmt.Errorf("%s out of range", name)
	}
	return v, nil
}

func parseMeta(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return map[string]any{}, nil
	}
	if len(s) > maxMetaLen {
		return nil, fmt.Errorf("metadata too large")
	}
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return nil, fmt.Errorf("metadata not json object")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func ParseNLUResponse(content string) (resp *model.NLUResponse, err error) {
	// panic safety
	defer func() {
		if r := recover(); r != nil {
			logx.Error().Str("component", "nlu_parser").Msgf("panic recovered: %v", r)
			err = errx.New(fmt.Errorf("nlu parser panic"), http.StatusInternalServerError, errx.SystemErrorMessage)
			resp = nil
		}
	}()

	// content length guard
	truncated := false
	if len(content) > maxContentLen {
		logx.Warn().
			Str("component", "nlu_parser").
			Int("max_len", maxContentLen).
			Int("orig_len", len(content)).
			Msg("content truncated due to size limit")
		content = content[:maxContentLen]
		truncated = true
	}
	// honor completion delimiter if present
	if idx := strings.Index(content, endDelim); idx >= 0 {
		content = content[:idx]
	}

	resp = &model.NLUResponse{
		Intents:         []model.Intent{},
		Entities:        []model.Entity{},
		Languages:       []model.Language{},
		Sentiment:       model.Sentiment{},
		ImportanceScore: 0,
		PrimaryIntent:   "",
		PrimaryLanguage: "",
		Metadata:        map[string]any{"parser": "lite"},
		ParsingMetadata: map[string]any{},
		Timestamp:       time.Now().UTC(),
	}

	addErr := func(msg string) {
		if resp.ParsingMetadata == nil {
			resp.ParsingMetadata = make(map[string]any)
		}
		v, _ := resp.ParsingMetadata["parsing_errors"].([]string)
		v = append(v, msg)
		resp.ParsingMetadata["parsing_errors"] = v
	}

	if truncated {
		resp.ParsingMetadata["truncated"] = true
	}

	records := strings.Split(content, recDelim)
	processed := 0
	for _, rec := range records {
		if processed >= maxRecords {
			resp.ParsingMetadata["records_capped"] = true
			logx.Warn().
				Str("component", "nlu_parser").
				Int("max_records", maxRecords).
				Msg("record processing capped")
			break
		}
		rec = strings.TrimSpace(rec)
		if rec == "" || rec == endDelim {
			continue
		}
		processed++

		rt, rerr := parseRawTuple(rec)
		if rerr != nil {
			addErr(fmt.Sprintf("bad_record: %s", safeSnippet(rec)))
			continue
		}

		switch rt.Type {
		case "intent":
			if len(rt.Parts) < 4 {
				addErr("intent: insufficient parts")
				continue
			}
			name := strings.TrimSpace(rt.Parts[1])
			if err := mustValidUTF8(name, "intent.name"); err != nil || name == "" {
				addErr("intent: invalid name utf8")
				continue
			}
			conf, err := parseFloatInRange(rt.Parts[2], "intent.confidence", 0, 1)
			if err != nil {
				addErr("intent: invalid confidence")
				continue
			}
			prio, err := parseFloatInRange(rt.Parts[3], "intent.priority", 0, 1)
			if err != nil {
				addErr("intent: invalid priority")
				continue
			}
			meta := map[string]any{}
			if len(rt.Parts) >= 5 {
				if m, err := parseMeta(rt.Parts[4]); err == nil {
					meta = m
				} else {
					addErr("intent: invalid metadata json")
				}
			}
			resp.Intents = append(resp.Intents, model.Intent{Name: name, Confidence: conf, Priority: prio, Metadata: meta})

		case "entity":
			if len(rt.Parts) < 4 {
				addErr("entity: insufficient parts")
				continue
			}
			etype := strings.TrimSpace(rt.Parts[1])
			val := strings.TrimSpace(rt.Parts[2])
			if err := mustValidUTF8(etype, "entity.type"); err != nil || etype == "" {
				addErr("entity: invalid type utf8")
				continue
			}
			if err := mustValidUTF8(val, "entity.value"); err != nil || val == "" {
				addErr("entity: invalid value utf8")
				continue
			}
			conf, err := parseFloatInRange(rt.Parts[3], "entity.confidence", 0, 1)
			if err != nil {
				addErr("entity: invalid confidence")
				continue
			}
			meta := map[string]any{}
			if len(rt.Parts) >= 5 {
				if m, err := parseMeta(rt.Parts[4]); err == nil {
					// validate known fields
					if pos := normalizeEntityPosition(m); len(pos) == 2 {
						// ok, keep as is and also set Position
					}
					meta = m
				} else {
					addErr("entity: invalid metadata json")
				}
			}
			e := model.Entity{Type: etype, Value: val, Confidence: conf, Metadata: meta}
			if pos := normalizeEntityPosition(meta); len(pos) == 2 {
				e.Position = pos
			}
			resp.Entities = append(resp.Entities, e)

		case "language":
			if len(rt.Parts) < 4 {
				addErr("language: insufficient parts")
				continue
			}
			code := strings.ToLower(strings.TrimSpace(rt.Parts[1]))
			if !isISO639_3(code) || mustValidUTF8(code, "lang.code") != nil {
				addErr("language: invalid code")
				continue
			}
			conf, err := parseFloatInRange(rt.Parts[2], "lang.confidence", 0, 1)
			if err != nil {
				addErr("language: invalid confidence")
				continue
			}
			isPrimary := strings.TrimSpace(rt.Parts[3]) == "1"
			meta := map[string]any{}
			if len(rt.Parts) >= 5 {
				if m, err := parseMeta(rt.Parts[4]); err == nil {
					// sanitize known fields
					sanitizeLanguageMeta(m)
					meta = m
				} else {
					addErr("language: invalid metadata json")
				}
			}
			resp.Languages = append(resp.Languages, model.Language{Code: code, Confidence: conf, IsPrimary: isPrimary, Metadata: meta})

		case "sentiment":
			if len(rt.Parts) < 3 {
				addErr("sentiment: insufficient parts")
				continue
			}
			label := strings.TrimSpace(rt.Parts[1])
			if err := mustValidUTF8(label, "sent.label"); err != nil || label == "" {
				addErr("sentiment: invalid label utf8")
				continue
			}
			conf, err := parseFloatInRange(rt.Parts[2], "sent.confidence", 0, 1)
			if err != nil {
				addErr("sentiment: invalid confidence")
				continue
			}
			meta := map[string]any{}
			if len(rt.Parts) >= 4 {
				if m, err := parseMeta(rt.Parts[3]); err == nil {
					sanitizeSentimentMeta(m)
					meta = m
				} else {
					addErr("sentiment: invalid metadata json")
				}
			}
			resp.Sentiment = model.Sentiment{Label: label, Confidence: conf, Metadata: meta}
		default:
			// ignore unknown type but record a hint
			addErr("unknown tuple type")
		}
	}

	// Derived fields
	// PrimaryIntent: highest confidence
	bestConf := -1.0
	for _, it := range resp.Intents {
		if it.Confidence > bestConf {
			bestConf = it.Confidence
			resp.PrimaryIntent = it.Name
		}
	}
	// PrimaryLanguage: first primary or highest confidence
	for _, l := range resp.Languages {
		if l.IsPrimary {
			resp.PrimaryLanguage = l.Code
			break
		}
	}
	if resp.PrimaryLanguage == "" {
		best := -1.0
		for _, l := range resp.Languages {
			if l.Confidence > best {
				best = l.Confidence
				resp.PrimaryLanguage = l.Code
			}
		}
	}
	// ImportanceScore: 0.6*confidence + 0.4*priority (primary intent)
	if len(resp.Intents) > 0 {
		conf := 0.0
		prio := 0.0
		for _, it := range resp.Intents {
			if it.Name == resp.PrimaryIntent {
				conf = it.Confidence
				prio = it.Priority
				break
			}
		}
		resp.ImportanceScore = conf*0.6 + prio*0.4
	}

	return resp, nil
}

// --- helpers ---

func safeSnippet(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxErrSnippet {
		return s
	}
	return s[:maxErrSnippet]
}

func isISO639_3(code string) bool {
	if len(code) != 3 {
		return false
	}
	for i := 0; i < 3; i++ {
		c := code[i]
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

func normalizeEntityPosition(meta map[string]any) []int {
	if meta == nil {
		return nil
	}
	raw, ok := meta["entity_position"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) != 2 {
		return nil
	}
	a, aok := arr[0].(float64)
	b, bok := arr[1].(float64)
	if !aok || !bok {
		return nil
	}
	start := int(a)
	end := int(b)
	if start < 0 || end < 0 || start > end {
		return nil
	}
	return []int{start, end}
}

func sanitizeLanguageMeta(m map[string]any) {
	if m == nil {
		return
	}
	if v, ok := m["detected_tokens"].(float64); ok {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			delete(m, "detected_tokens")
		} else {
			// keep as number; downstream can interpret as int
		}
	}
	if _, ok := m["script"].(string); !ok {
		// delete even if missing (no-op) or wrong type (cleanup)
		delete(m, "script")
	}
}

func sanitizeSentimentMeta(m map[string]any) {
	if m == nil {
		return
	}
	if v, ok := m["polarity"].(float64); ok {
		if v < -1 || v > 1 || math.IsNaN(v) || math.IsInf(v, 0) {
			delete(m, "polarity")
		}
	} else {
		// delete even if missing (no-op) or wrong type (cleanup)
		delete(m, "polarity")
	}
	if v, ok := m["subjectivity"].(float64); ok {
		if v < 0 || v > 1 || math.IsNaN(v) || math.IsInf(v, 0) {
			delete(m, "subjectivity")
		}
	} else {
		// delete even if missing (no-op) or wrong type (cleanup)
		delete(m, "subjectivity")
	}
}
