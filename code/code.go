package code

import (
	"github.com/thatrajaryan/web-server/common"
)

type LanguageEnum int

const (
    CPP LanguageEnum = iota
    JAVA
    JAVASCRIPT
    PYTHON
	GO
)

func (s LanguageEnum) String() string {
    return [...]string{"CPP", "JAVA", "JAVASCRIPT", "PYTHON", "GO"}[s]
}

type CodeBlock struct {
	Code string
	Language LanguageEnum
}

func (b *CodeBlock) Create(config map[string]interface{}) error {
	// Implementation to be added later
	return nil
}

func (b *CodeBlock) Connect(target common.Block) error {
	// Implementation to be added later
	return nil
}

func (b *CodeBlock) Update(config map[string]interface{}) error {
	return nil
}

func (b *CodeBlock) Delete() error {
	return nil
}

func (b *CodeBlock) Status() string {
	return "Active"
}

func (b *CodeBlock) Start() error {
	return nil
}

func (b *CodeBlock) Stop() error {
	return nil
}