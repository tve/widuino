package database

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	"testing"
)

func TestRsclient(t *testing.T) {
	format.UseStringerRepresentation = true
	RegisterFailHandler(Fail)
	RunSpecs(t, "Database Suite")
}
