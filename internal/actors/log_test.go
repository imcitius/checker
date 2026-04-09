// SPDX-License-Identifier: BUSL-1.1

package actors

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLogActor_Act(t *testing.T) {
	// Capture logrus output to verify the message is logged.
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(nil)

	actor := &LogActor{Message: "static message"}
	err := actor.Act("dynamic message passed to Act")

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "dynamic message passed to Act")
}

func TestLogActor_Act_EmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(nil)

	actor := &LogActor{}
	err := actor.Act("")

	assert.NoError(t, err)
}

func TestLogActor_Act_MultipleInvocations(t *testing.T) {
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(nil)

	actor := &LogActor{Message: "check"}
	assert.NoError(t, actor.Act("first"))
	assert.NoError(t, actor.Act("second"))

	output := buf.String()
	assert.Contains(t, output, "first")
	assert.Contains(t, output, "second")
}
