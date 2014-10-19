package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"

	// test logging
	//"fmt"
	//"github.com/golang/glog"
	//"os"

	"testing"
)

func TestSuite(t *testing.T) {
	//fmt.Fprintln(os.Stdout, "STDOUT")
	//fmt.Fprintln(os.Stderr, "STDERR")
	//glog.Error("glog.Error")
	//glog.Warning("glog.Warning")
	//glog.Info("glog.Info")

	format.UseStringerRepresentation = true
	RegisterFailHandler(Fail)
	RunSpecs(t, "NodeGW Suite")
}
