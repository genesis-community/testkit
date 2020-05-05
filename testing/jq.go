package testing

import (
	"bytes"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
)

type jq struct {
	query     string
	variables []string
	values    []interface{}
}

func (c jq) Run(input interface{}) []byte {
	query, err := gojq.Parse(c.query)
	Expect(err).ToNot(HaveOccurred())

	code, err := gojq.Compile(
		query,
		gojq.WithVariables(c.variables),
	)
	Expect(err).ToNot(HaveOccurred())

	var buf bytes.Buffer

	iter := code.Run(input, c.values...)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			Expect(err).ToNot(HaveOccurred())
		}
		out, err := yaml.Marshal(v)
		Expect(err).ToNot(HaveOccurred())
		_, err = buf.Write(out)
		Expect(err).ToNot(HaveOccurred())
	}
	return buf.Bytes()
}
