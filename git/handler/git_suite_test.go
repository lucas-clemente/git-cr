package handler_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGitHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Git Handler Suite")
}
