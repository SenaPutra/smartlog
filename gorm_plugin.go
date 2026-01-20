package smartlog

import (
	"encoding/json"
	"sort"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GormResultLogPlugin is a GORM plugin to log query results.
type GormResultLogPlugin struct {
	logger *zap.Logger
	cfg    GormConfig
}

// NewGormResultLogPlugin creates a new GormResultLogPlugin.
func NewGormResultLogPlugin(logger *zap.Logger, cfg GormConfig) *GormResultLogPlugin {
	return &GormResultLogPlugin{logger: logger, cfg: cfg}
}

// Name returns the name of the plugin.
func (p *GormResultLogPlugin) Name() string {
	return "GormResultLogPlugin"
}

// Initialize initializes the plugin.
func (p *GormResultLogPlugin) Initialize(db *gorm.DB) error {
	if !p.cfg.LogQueryResult {
		return nil
	}

	return db.Callback().Query().After("gorm:query").Register("smartlog:log_result", p.logResult)
}

func (p *GormResultLogPlugin) logResult(db *gorm.DB) {
	ctx := db.Statement.Context
	logger := p.logger
	if ctx != nil {
		if ctxLogger, ok := ctx.Value(LoggerKey).(*zap.Logger); ok {
			logger = ctxLogger
		}
	}

	resultJSON, err := json.Marshal(db.Statement.Dest)
	if err != nil {
		logger.Warn("Failed to marshal GORM query result", zap.Error(err))
		return
	}

	// Truncate if the result is larger than the max bytes
	if p.cfg.LogResultMaxBytes > 0 && len(resultJSON) > p.cfg.LogResultMaxBytes {
		var data interface{}
		if err := json.Unmarshal(resultJSON, &data); err != nil {
			logger.Warn("Failed to unmarshal GORM query result for truncation", zap.Error(err))
			// Fallback to simple truncation if unmarshaling fails
			resultJSON = resultJSON[:p.cfg.LogResultMaxBytes]
		} else {
			// Create a new map to hold the truncated result
			truncatedMap := make(map[string]interface{})
			keys := make([]string, 0)
			if m, ok := data.(map[string]interface{}); ok {
				for key := range m {
					keys = append(keys, key)
				}
				sort.Strings(keys) // Sort the keys
			}

			// Add fields one by one and check the size
			currentSize := 2 // for '{}'
			if id, ok := data.(map[string]interface{})["ID"]; ok {
				truncatedMap["ID"] = id
				fieldJSON, _ := json.Marshal(map[string]interface{}{"ID": id})
				currentSize += len(fieldJSON) - 1
			}

			for _, key := range keys {
				if key == "ID" {
					continue // Already added
				}
				value := data.(map[string]interface{})[key]
				fieldJSON, _ := json.Marshal(map[string]interface{}{key: value})
				if currentSize+len(fieldJSON)-1 > p.cfg.LogResultMaxBytes {
					break
				}
				truncatedMap[key] = value
				currentSize += len(fieldJSON) - 1
			}
			resultJSON, _ = json.Marshal(truncatedMap)
		}
	}

	logger.Debug("GORM Query Result", zap.ByteString("result", resultJSON))
}
