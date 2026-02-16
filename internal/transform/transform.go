package transform

import "github.com/bigbes/lua-amalgamate/internal/config"

type Transformer interface {
	Transform(source []byte) ([]byte, error)
}

func BuildPipeline(cfg config.TransformConfig) []Transformer {
	var pipeline []Transformer
	if cfg.Minify {
		pipeline = append(pipeline, &minifyTransformer{})
	} else {
		if cfg.StripShebang {
			pipeline = append(pipeline, &shebangTransformer{})
		}
		if cfg.RemoveComments {
			pipeline = append(pipeline, &commentTransformer{})
		}
		if cfg.RemoveEmptyLines {
			pipeline = append(pipeline, &emptyLineTransformer{})
		}
	}
	return pipeline
}
