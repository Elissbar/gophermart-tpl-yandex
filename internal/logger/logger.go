package logger

import "go.uber.org/zap"

func NewLogger(logLevel string) (*zap.Logger, error) {
    lvl, err := zap.ParseAtomicLevel(logLevel)
    if err != nil {
        return zap.NewNop(), err
    }

    cfg := zap.NewProductionConfig()
    cfg.Level = lvl

    return cfg.Build()
}

func NewSugaredLogger(logLevel string) (*zap.SugaredLogger, error) {
    logger, err := NewLogger(logLevel)
    if err != nil {
        return logger.Sugar(), err
    }
    return logger.Sugar(), nil
}
