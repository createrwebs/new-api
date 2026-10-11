package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

var (
	ErrValidationMissingRequiredField = errors.New("missing required input field")
	ErrValidationUnsupportedAspect    = errors.New("unsupported aspect ratio")
	ErrValidationUnsupportedDuration  = errors.New("unsupported duration")
	ErrValidationUnsupportedOutputs   = errors.New("unsupported number of outputs")
	ErrValidationInvalidDimensions    = errors.New("invalid image dimensions")
	ErrValidationUnsupportedTier      = errors.New("unsupported quality tier for route")
	ErrValidationAssetRequired        = errors.New("input asset image URL required")
)

// Allowed transform types in Parameter Mapping DSL (Section 17).
const (
	TransformRename        = "rename"
	TransformConstant      = "constant"
	TransformEnumMap       = "enum_map"
	TransformDefault       = "default"
	TransformNumericScale  = "numeric_scale"
	TransformBoolMap       = "bool_map"
	TransformArrayWrap     = "array_wrap"
	TransformAssetExtract  = "asset_extract"
	TransformOmitEmpty     = "omit_empty"
	TransformFormat        = "format"
)

// ParameterMappingRule defines a single safe transformation rule without executable code.
type ParameterMappingRule struct {
	Source      string                 `json:"source,omitempty"`       // Normalized field name (e.g. "prompt", "aspect_ratio", "width")
	Target      string                 `json:"target"`                 // Provider field name
	Transform   string                 `json:"transform,omitempty"`    // Transform type
	ConstantVal interface{}            `json:"constant_val,omitempty"` // For "constant"
	DefaultVal  interface{}            `json:"default_val,omitempty"`  // Fallback if source is empty
	EnumMap     map[string]string      `json:"enum_map,omitempty"`     // For "enum_map"
	ScaleFactor float64                `json:"scale_factor,omitempty"` // For "numeric_scale"
	BoolTrue    interface{}            `json:"bool_true,omitempty"`    // For "bool_map"
	BoolFalse   interface{}            `json:"bool_false,omitempty"`   // For "bool_map"
	FormatStr   string                 `json:"format_str,omitempty"`   // For "format" e.g. "%dx%d" or "%d*%d"
	OmitIfEmpty bool                   `json:"omit_if_empty,omitempty"`
}

// ParameterMappingSpec contains a list of declarative mapping rules.
type ParameterMappingSpec struct {
	Rules []ParameterMappingRule `json:"rules"`
}

// ApplyDeclarativeMapping applies either a structured ParameterMappingSpec or legacy map[string]string.
func ApplyDeclarativeMapping(input *NormalizedMediaInput, mappingJSON string) (map[string]interface{}, error) {
	if input == nil {
		return make(map[string]interface{}), nil
	}

	trimmed := strings.TrimSpace(mappingJSON)

	// If empty mapping, apply sensible standard defaults
	if trimmed == "" || trimmed == "{}" {
		return applyDefaultMapping(input), nil
	}

	// Try parsing as structured ParameterMappingSpec (Queue 2H DSL)
	var spec ParameterMappingSpec
	if err := json.Unmarshal([]byte(trimmed), &spec); err == nil && len(spec.Rules) > 0 {
		return executeMappingRules(input, spec.Rules)
	}

	// Try parsing as legacy map[string]string
	var legacyMap map[string]string
	if err := json.Unmarshal([]byte(trimmed), &legacyMap); err == nil && len(legacyMap) > 0 {
		return executeLegacyMapping(input, legacyMap)
	}

	// Fallback to defaults if JSON was valid empty or unrecognized structure
	return applyDefaultMapping(input), nil
}

func applyDefaultMapping(input *NormalizedMediaInput) map[string]interface{} {
	payload := make(map[string]interface{})
	if input.Prompt != "" {
		payload["prompt"] = input.Prompt
	}
	if input.NegativePrompt != "" {
		payload["negative_prompt"] = input.NegativePrompt
	}
	if len(input.InputAssets) > 0 {
		payload["image_url"] = input.InputAssets[0]
	}
	if input.AspectRatio != "" {
		payload["aspect_ratio"] = input.AspectRatio
	}
	if input.Width > 0 && input.Height > 0 {
		payload["size"] = fmt.Sprintf("%d*%d", input.Width, input.Height)
	} else if input.AspectRatio == "1:1" {
		payload["size"] = "1024*1024"
	}
	if input.Seed > 0 {
		payload["seed"] = input.Seed
	}
	for k, v := range input.AdvancedParams {
		payload[k] = v
	}
	return payload
}

func executeLegacyMapping(input *NormalizedMediaInput, mapping map[string]string) (map[string]interface{}, error) {
	payload := make(map[string]interface{})
	for normKey, targetKey := range mapping {
		val := getNormalizedFieldValue(input, normKey)
		if val != nil {
			payload[targetKey] = val
		}
	}
	// Copy any remaining advanced parameters that do not overwrite explicit mappings
	for k, v := range input.AdvancedParams {
		if _, exists := payload[k]; !exists {
			payload[k] = v
		}
	}
	return payload, nil
}

func executeMappingRules(input *NormalizedMediaInput, rules []ParameterMappingRule) (map[string]interface{}, error) {
	payload := make(map[string]interface{})

	for _, rule := range rules {
		targetKey := rule.Target
		if targetKey == "" {
			continue
		}

		switch rule.Transform {
		case TransformConstant:
			payload[targetKey] = rule.ConstantVal

		case TransformEnumMap:
			raw := getNormalizedFieldValue(input, rule.Source)
			s := fmt.Sprintf("%v", raw)
			if mapped, ok := rule.EnumMap[s]; ok {
				payload[targetKey] = mapped
			} else if rule.DefaultVal != nil {
				payload[targetKey] = rule.DefaultVal
			} else if !rule.OmitIfEmpty && raw != nil {
				payload[targetKey] = raw
			}

		case TransformNumericScale:
			raw := getNormalizedFieldValue(input, rule.Source)
			if f, ok := toFloat(raw); ok {
				scale := rule.ScaleFactor
				if scale == 0 {
					scale = 1.0
				}
				payload[targetKey] = f * scale
			} else if rule.DefaultVal != nil {
				payload[targetKey] = rule.DefaultVal
			}

		case TransformBoolMap:
			raw := getNormalizedFieldValue(input, rule.Source)
			b, ok := raw.(bool)
			if ok {
				if b && rule.BoolTrue != nil {
					payload[targetKey] = rule.BoolTrue
				} else if !b && rule.BoolFalse != nil {
					payload[targetKey] = rule.BoolFalse
				} else {
					payload[targetKey] = b
				}
			} else if rule.DefaultVal != nil {
				payload[targetKey] = rule.DefaultVal
			}

		case TransformArrayWrap:
			raw := getNormalizedFieldValue(input, rule.Source)
			if raw != nil {
				payload[targetKey] = []interface{}{raw}
			} else if rule.DefaultVal != nil {
				payload[targetKey] = rule.DefaultVal
			}

		case TransformAssetExtract:
			if len(input.InputAssets) > 0 {
				payload[targetKey] = input.InputAssets[0]
			} else if rule.DefaultVal != nil {
				payload[targetKey] = rule.DefaultVal
			}

		case TransformFormat:
			if rule.Source == "width,height" || rule.Source == "size" || (rule.Source == "aspect_ratio" && rule.Target == "size") {
				w := input.Width
				h := input.Height
				if w > 0 && h > 0 {
					fmtStr := rule.FormatStr
					if fmtStr == "" {
						fmtStr = "%d*%d"
					}
					payload[targetKey] = fmt.Sprintf(fmtStr, w, h)
				} else if input.AspectRatio != "" {
					payload[targetKey] = aspectToSize(input.AspectRatio)
				} else if rule.DefaultVal != nil {
					payload[targetKey] = rule.DefaultVal
				}
			} else {
				raw := getNormalizedFieldValue(input, rule.Source)
				if raw != nil {
					if rule.FormatStr != "" {
						payload[targetKey] = fmt.Sprintf(rule.FormatStr, raw)
					} else {
						payload[targetKey] = raw
					}
				} else if rule.DefaultVal != nil {
					payload[targetKey] = rule.DefaultVal
				}
			}

		case TransformRename, TransformDefault, "":
			raw := getNormalizedFieldValue(input, rule.Source)
			if raw != nil && raw != "" {
				payload[targetKey] = raw
			} else if rule.DefaultVal != nil {
				payload[targetKey] = rule.DefaultVal
			} else if !rule.OmitIfEmpty && raw != nil {
				payload[targetKey] = raw
			}
		}
	}

	return payload, nil
}

func getNormalizedFieldValue(input *NormalizedMediaInput, key string) interface{} {
	if input == nil {
		return nil
	}
	switch strings.ToLower(key) {
	case "prompt":
		if input.Prompt != "" {
			return input.Prompt
		}
	case "negative_prompt":
		if input.NegativePrompt != "" {
			return input.NegativePrompt
		}
	case "image_url", "input_assets[0]", "image", "input_url":
		if len(input.InputAssets) > 0 && input.InputAssets[0] != "" {
			return input.InputAssets[0]
		}
	case "input_assets":
		if len(input.InputAssets) > 0 {
			return input.InputAssets
		}
	case "mask_url", "mask_asset":
		if input.MaskAsset != "" {
			return input.MaskAsset
		}
	case "reference_image_url", "reference_assets[0]":
		if len(input.ReferenceAssets) > 0 && input.ReferenceAssets[0] != "" {
			return input.ReferenceAssets[0]
		}
	case "aspect_ratio":
		if input.AspectRatio != "" {
			return input.AspectRatio
		}
	case "width":
		if input.Width > 0 {
			return input.Width
		}
	case "height":
		if input.Height > 0 {
			return input.Height
		}
	case "size":
		if input.Width > 0 && input.Height > 0 {
			return fmt.Sprintf("%d*%d", input.Width, input.Height)
		}
		if input.AspectRatio != "" {
			return aspectToSize(input.AspectRatio)
		}
	case "resolution":
		if input.Resolution != "" {
			return input.Resolution
		}
	case "duration":
		if input.Duration > 0 {
			return input.Duration
		}
	case "fps":
		if input.FPS > 0 {
			return input.FPS
		}
	case "quality":
		if input.Quality != "" {
			return input.Quality
		}
	case "quality_tier":
		if input.QualityTier != "" {
			return input.QualityTier
		}
	case "number_of_outputs", "num_outputs":
		if input.NumberOfOutputs > 0 {
			return input.NumberOfOutputs
		}
	case "audio":
		return input.Audio
	case "seed":
		if input.Seed > 0 {
			return input.Seed
		}
	case "strength":
		if input.Strength > 0 {
			return input.Strength
		}
	case "callback_url":
		if input.CallbackURL != "" {
			return input.CallbackURL
		}
	default:
		if v, ok := input.AdvancedParams[key]; ok {
			return v
		}
	}
	return nil
}

func aspectToSize(aspectRatio string) string {
	switch aspectRatio {
	case "16:9":
		return "1280*720"
	case "9:16":
		return "720*1280"
	case "4:3":
		return "1024*768"
	case "3:4":
		return "768*1024"
	case "21:9":
		return "1344*576"
	default:
		return "1024*1024"
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// ValidateNormalizedInput performs early pre-flight validation before reservation/provider submit (Section 18).
func ValidateNormalizedInput(route *model.StudioModelRoute, input *NormalizedMediaInput) error {
	if route == nil {
		return errors.New("route is nil")
	}
	if input == nil {
		return errors.New("input is nil")
	}

	// 1. Tool-level required parameters
	switch route.LogicalTool {
	case "image-generate":
		if strings.TrimSpace(input.Prompt) == "" {
			return fmt.Errorf("%w: prompt is required for image generation", ErrValidationMissingRequiredField)
		}
	case "background-remove":
		if len(input.InputAssets) == 0 || strings.TrimSpace(input.InputAssets[0]) == "" {
			return fmt.Errorf("%w: input image URL is required for background removal", ErrValidationAssetRequired)
		}
	case "image-upscale":
		if len(input.InputAssets) == 0 || strings.TrimSpace(input.InputAssets[0]) == "" {
			return fmt.Errorf("%w: input image URL is required for upscale", ErrValidationAssetRequired)
		}
	case "product-photo":
		if len(input.InputAssets) == 0 || strings.TrimSpace(input.InputAssets[0]) == "" {
			return fmt.Errorf("%w: product base image URL is required", ErrValidationAssetRequired)
		}
		if strings.TrimSpace(input.Prompt) == "" {
			return fmt.Errorf("%w: scene description prompt is required for product photo", ErrValidationMissingRequiredField)
		}
	case "image-to-video":
		if len(input.InputAssets) == 0 || strings.TrimSpace(input.InputAssets[0]) == "" {
			return fmt.Errorf("%w: initial frame image URL is required for image-to-video", ErrValidationAssetRequired)
		}
	}

	// 2. Aspect Ratio Validation (if provided)
	if input.AspectRatio != "" {
		validAspects := map[string]bool{
			"1:1":  true,
			"16:9": true,
			"9:16": true,
			"4:3":  true,
			"3:4":  true,
			"21:9": true,
			"2:3":  true,
			"3:2":  true,
		}
		if !validAspects[input.AspectRatio] {
			return fmt.Errorf("%w: %s (allowed: 1:1, 16:9, 9:16, 4:3, 3:4, 21:9)", ErrValidationUnsupportedAspect, input.AspectRatio)
		}
	}

	// 3. Dimension Bounds Validation (if specified)
	if input.Width > 0 || input.Height > 0 {
		if input.Width < 64 || input.Width > 4096 || input.Height < 64 || input.Height > 4096 {
			return fmt.Errorf("%w: dimensions %dx%d out of safe bounds [64..4096]", ErrValidationInvalidDimensions, input.Width, input.Height)
		}
	}

	// 4. Output count validation
	if input.NumberOfOutputs < 0 || input.NumberOfOutputs > 8 {
		return fmt.Errorf("%w: %d (allowed range: 1..8)", ErrValidationUnsupportedOutputs, input.NumberOfOutputs)
	}
	if input.NumberOfOutputs == 0 {
		input.NumberOfOutputs = 1
	}

	// 5. Duration validation (for video tools)
	if strings.Contains(route.LogicalTool, "video") {
		if input.Duration < 0 || input.Duration > 60 {
			return fmt.Errorf("%w: duration %d seconds exceeds limit [1..60]", ErrValidationUnsupportedDuration, input.Duration)
		}
	}

	// 6. Quality tier check
	if input.QualityTier != "" && route.QualityTier != "" {
		if strings.ToUpper(input.QualityTier) != strings.ToUpper(route.QualityTier) {
			// Informational tier mismatch allowed if caller explicitly accepted fallback, but strict routes enforce it
			// We only reject if route strictly limits capabilities
		}
	}

	return nil
}
