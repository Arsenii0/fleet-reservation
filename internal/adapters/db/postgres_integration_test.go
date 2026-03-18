package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

// TODO ArsenP: Implement
type DBTestSuite struct {
	suite.Suite
	container testcontainers.Container
	adapter   *Adapter
}

func (s *DBTestSuite) SetupSuite() {

}

func (s *DBTestSuite) TearDownSuite() {
	err := s.container.Terminate(context.Background())
	s.Require().NoError(err)
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}
