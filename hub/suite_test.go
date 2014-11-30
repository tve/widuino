package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	"testing"
)

func TestSuite(t *testing.T) {
	format.UseStringerRepresentation = true
	RegisterFailHandler(Fail)
	RunSpecs(t, "NodeGW Suite")
}
