// Copyright © 2016 Asteris, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package user_test

import (
	"fmt"
	"math"
	os "os/user"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/asteris-llc/converge/helpers/fakerenderer"
	"github.com/asteris-llc/converge/resource"
	"github.com/asteris-llc/converge/resource/user"
	"github.com/fgrid/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	currUser          *os.User
	currUsername      string
	currUID           string
	currGroup         *os.Group
	currGroupName     string
	currGID           string
	existingGroup     *os.Group
	existingGroupName string
	existingGID       string
	existingUser      *os.User
	existingUID       string
	tempUsername      []string
	fakeUsername      string
	fakeUID           string
	tempGroupName     []string
	fakeGroupName     string
	fakeGID           string
	err               error
)

const (
	// minGID designates the smallest valid GID
	// At a minimum, 0-32676 is valid
	minGID = 0

	// maxGID designates the largest valid GID
	// At a minimum, 0-32676 is valid
	maxGID = math.MaxInt16

	// minUID designates the smallest valid UID
	// At a minimum, 0-32676 is valid
	minUID = 0

	// maxUID designates the largest valid UID
	// At a minimum, 0-32676 is valid
	maxUID = math.MaxInt16
)

func init() {
	currUser, err = os.Current()
	if err != nil {
		panic(err)
	}

	currUsername = currUser.Username
	currUID = currUser.Uid

	currGID = currUser.Gid
	currGroup, err = os.LookupGroupId(currGID)
	if err != nil {
		panic(err)
	}
	currGroupName = currGroup.Name

	fakeUID, err = setFakeUid()
	if err != nil {
		panic(err)
	}
	fakeGID, err = setFakeGid()
	if err != nil {
		panic(err)
	}

	tempUsername = strings.Split(uuid.NewV4().String(), "-")
	fakeUsername = strings.Join(tempUsername[0:], "")
	tempGroupName = strings.Split(uuid.NewV4().String(), "-")
	fakeGroupName = strings.Join(tempUsername[0:], "")

	existingGID, err = setGid()
	if err != nil {
		panic(err)
	}
	existingGroup, err = os.LookupGroupId(existingGID)
	if err != nil {
		panic(err)
	}
	existingGroupName = existingGroup.Name

	existingUID, err = setUid()
	if err != nil {
		panic(err)
	}
	existingUser, err = os.LookupId(existingUID)
	if err != nil {
		panic(err)
	}
}

// TestUserInterface tests that User is properly implemented
func TestUserInterface(t *testing.T) {
	t.Parallel()

	assert.Implements(t, (*resource.Task)(nil), new(user.User))
}

// TestCheck tests the possible cases Check handles
func TestCheck(t *testing.T) {
	t.Parallel()

	t.Run("state=present", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.State = user.StatePresent

		t.Run("add tests", func(t *testing.T) {
			t.Run("add user", func(t *testing.T) {
				u.Username = fakeUsername
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.NoError(t, err)
					assert.Equal(t, "add user", status.Messages()[0])
					assert.Equal(t, resource.StatusWillChange, status.StatusCode())
					assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), status.Diffs()["username"].Original())
					assert.Equal(t, u.Username, status.Diffs()["username"].Current())
					assert.True(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})

			t.Run("cannot add user", func(t *testing.T) {
				u.Username = fakeUsername
				u.GroupName = fakeGroupName
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.EqualError(t, err, fmt.Sprintf("cannot add user %s: group %s does not exist", u.Username, u.GroupName))
					assert.Equal(t, resource.StatusCantChange, status.StatusCode())
					assert.True(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})
		})

		t.Run("modify tests", func(t *testing.T) {
			t.Run("no modifications", func(t *testing.T) {
				u.Username = currUsername
				u.GroupName = "" // clear this field set from previous t.Run
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.NoError(t, err)
					assert.Equal(t, resource.StatusNoChange, status.StatusCode())
					assert.Equal(t, fmt.Sprintf("no modifications indicated for user %s", u.Username), status.Messages()[0])
					assert.False(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})

			t.Run("cannot modify user", func(t *testing.T) {
				u.Username = currUsername
				u.GroupName = fakeGroupName
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.EqualError(t, err, fmt.Sprintf("cannot modify user %s: group %s does not exist", u.Username, u.GroupName))
					assert.Equal(t, resource.StatusCantChange, status.StatusCode())
					assert.True(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})

			t.Run("modify user", func(t *testing.T) {
				u.Username = currUsername
				u.NewUsername = fakeUsername
				u.GroupName = "" // clear this field set from previous t.Run
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.NoError(t, err)
					assert.Equal(t, resource.StatusWillChange, status.StatusCode())
					assert.Equal(t, "modify user", status.Messages()[0])
					assert.Equal(t, u.Username, status.Diffs()["username"].Original())
					assert.Equal(t, u.NewUsername, status.Diffs()["username"].Current())
					assert.True(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})
		})
	})

	t.Run("state=absent", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.State = user.StateAbsent

		t.Run("uid not provided", func(t *testing.T) {
			t.Run("no delete-user does not exist", func(t *testing.T) {
				u.Username = fakeUsername
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.NoError(t, err)
					assert.Equal(t, resource.StatusNoChange, status.StatusCode())
					assert.Equal(t, fmt.Sprintf("user %s does not exist", u.Username), status.Messages()[0])
					assert.False(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})

			t.Run("delete user", func(t *testing.T) {
				u.Username = currUsername
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.NoError(t, err)
					assert.Equal(t, resource.StatusWillChange, status.StatusCode())
					assert.Equal(t, fmt.Sprintf("user %s", u.Username), status.Diffs()["user"].Original())
					assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), status.Diffs()["user"].Current())
					assert.True(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})
		})

		t.Run("uid provided", func(t *testing.T) {
			t.Run("no delete-user name and uid do not exist", func(t *testing.T) {
				u.Username = fakeUsername
				u.UID = fakeUID
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.NoError(t, err)
					assert.Equal(t, resource.StatusNoChange, status.StatusCode())
					assert.Equal(t, fmt.Sprintf("user %s and uid %s do not exist", u.Username, u.UID), status.Messages()[0])
					assert.False(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})

			t.Run("no delete-user name does not exist", func(t *testing.T) {
				u.Username = fakeUsername
				u.UID = currUID
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.EqualError(t, err, fmt.Sprintf("cannot delete user %s with uid %s: user does not exist", u.Username, u.UID))
					assert.Equal(t, resource.StatusCantChange, status.StatusCode())
					assert.True(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})

			t.Run("no delete-user uid does not exist", func(t *testing.T) {
				u.Username = currUsername
				u.UID = fakeUID
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.EqualError(t, err, fmt.Sprintf("cannot delete user %s with uid %s: uid does not exist", u.Username, u.UID))
					assert.Equal(t, resource.StatusCantChange, status.StatusCode())
					assert.True(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})

			t.Run("no delete-user name and uid belong to different users", func(t *testing.T) {
				u.Username = currUsername
				u.UID = existingUID
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.EqualError(t, err, fmt.Sprintf("cannot delete user %s with uid %s: user and uid belong to different users", u.Username, u.UID))
					assert.Equal(t, resource.StatusCantChange, status.StatusCode())
					assert.True(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})

			t.Run("delete user with uid", func(t *testing.T) {
				u.Username = currUsername
				u.UID = currUID
				status, err := u.Check(fakerenderer.New())

				if runtime.GOOS == "linux" {
					assert.NoError(t, err)
					assert.Equal(t, resource.StatusWillChange, status.StatusCode())
					assert.Equal(t, fmt.Sprintf("user %s with uid %s", u.Username, u.UID), status.Diffs()["user"].Original())
					assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), status.Diffs()["user"].Current())
					assert.True(t, status.HasChanges())
				} else {
					assert.EqualError(t, err, "user: not supported on this system")
				}
			})
		})
	})

	t.Run("state unknown", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.Username = currUsername
		u.UID = currUID
		u.State = "test"
		status, err := u.Check(fakerenderer.New())

		if runtime.GOOS == "linux" {
			assert.EqualError(t, err, fmt.Sprintf("user: unrecognized state %s", u.State))
			assert.Equal(t, resource.StatusFatal, status.StatusCode())
		} else {
			assert.EqualError(t, err, "user: not supported on this system")
		}
	})

}

// TestApply tests all possible cases Apply handles
func TestApply(t *testing.T) {
	t.Parallel()

	t.Run("state=present", func(t *testing.T) {
		t.Run("add tests", func(t *testing.T) {
			t.Run("add user", func(t *testing.T) {
				usr := &os.User{
					Username: fakeUsername,
				}
				m := &MockSystem{}
				d := &MockDiff{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.State = user.StatePresent
				options := user.AddUserOptions{}

				m.On("Lookup", u.Username).Return(usr, os.UnknownUserError(""))
				d.On("DiffAdd", u.Status).Return(options, nil)
				m.On("AddUser", u.Username, &options).Return(nil)
				status, err := u.Apply()

				m.AssertCalled(t, "AddUser", u.Username, &options)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("added user %s", u.Username), status.Messages()[0])
			})

			t.Run("will not attempt to add", func(t *testing.T) {
				usr := &os.User{
					Username: fakeUsername,
				}
				grp := &os.Group{
					Name: fakeGroupName,
				}
				m := &MockSystem{}
				d := &MockDiff{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.GroupName = grp.Name
				u.State = user.StatePresent
				options := user.AddUserOptions{}
				optErr := fmt.Sprintf("group %s does not exist", u.GroupName)

				m.On("Lookup", u.Username).Return(usr, os.UnknownUserError(""))
				m.On("LookupGroup", u.GroupName).Return(grp, os.UnknownGroupError(""))
				d.On("DiffAdd", u.Status).Return(nil, optErr)
				m.On("AddUser", u.Username, &options).Return(nil)
				status, err := u.Apply()

				m.AssertNotCalled(t, "AddUser", u.Username, &options)
				assert.EqualError(t, err, fmt.Sprintf("will not attempt to add user %s: %s", u.Username, optErr))
				assert.Equal(t, resource.StatusCantChange, status.StatusCode())
			})

			t.Run("error adding user", func(t *testing.T) {
				usr := &os.User{
					Username: fakeUsername,
				}
				m := &MockSystem{}
				d := &MockDiff{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.State = user.StatePresent
				options := user.AddUserOptions{}

				m.On("Lookup", u.Username).Return(usr, os.UnknownUserError(""))
				d.On("DiffAdd", u.Status).Return(options, nil)
				m.On("AddUser", u.Username, &options).Return(fmt.Errorf(""))
				status, err := u.Apply()

				m.AssertCalled(t, "AddUser", u.Username, &options)
				assert.EqualError(t, err, "user add: ")
				assert.Equal(t, resource.StatusFatal, status.StatusCode())
				assert.Equal(t, fmt.Sprintf("error adding user %s", u.Username), status.Messages()[0])
			})
		})

		t.Run("modify tests", func(t *testing.T) {
			t.Run("modify user", func(t *testing.T) {
				usr := &os.User{
					Username: currUsername,
				}
				m := &MockSystem{}
				d := &MockDiff{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.Name = "test"
				u.State = user.StatePresent
				options := user.ModUserOptions{Comment: u.Name}

				m.On("Lookup", u.Username).Return(usr, nil)
				d.On("DiffMod", u.Status, currUser).Return(options, nil)
				m.On("ModUser", u.Username, &options).Return(nil)
				status, err := u.Apply()

				m.AssertCalled(t, "ModUser", u.Username, &options)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("modified user %s", u.Username), status.Messages()[0])
			})

			t.Run("will not attempt to modify", func(t *testing.T) {
				usr := &os.User{
					Username: currUsername,
				}
				grp := &os.Group{
					Name: fakeGroupName,
				}
				m := &MockSystem{}
				d := &MockDiff{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.GroupName = grp.Name
				u.State = user.StatePresent
				options := user.ModUserOptions{}
				optErr := fmt.Sprintf("group %s does not exist", u.GroupName)

				m.On("Lookup", u.Username).Return(usr, nil)
				m.On("LookupGroup", u.GroupName).Return(grp, os.UnknownGroupError(""))
				d.On("DiffMod", u.Status, currUser).Return(nil, optErr)
				m.On("ModUser", u.Username, &options).Return(nil)
				status, err := u.Apply()

				m.AssertNotCalled(t, "ModUser", u.Username, &options)
				assert.EqualError(t, err, fmt.Sprintf("will not attempt to modify user %s: %s", u.Username, optErr))
				assert.Equal(t, resource.StatusCantChange, status.StatusCode())
			})

			t.Run("error modifying user", func(t *testing.T) {
				usr := &os.User{
					Username: currUsername,
				}
				m := &MockSystem{}
				d := &MockDiff{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.Name = "test"
				u.State = user.StatePresent
				options := user.ModUserOptions{Comment: u.Name}

				m.On("Lookup", u.Username).Return(usr, nil)
				d.On("DiffMod", u.Status, currUser).Return(options, nil)
				m.On("ModUser", u.Username, &options).Return(fmt.Errorf(""))
				status, err := u.Apply()

				m.AssertCalled(t, "ModUser", u.Username, &options)
				assert.EqualError(t, err, "user modify: ")
				assert.Equal(t, resource.StatusFatal, status.StatusCode())
				assert.Equal(t, fmt.Sprintf("error modifying user %s", u.Username), status.Messages()[0])
			})
		})
	})

	t.Run("state=absent", func(t *testing.T) {
		t.Run("uid not provided", func(t *testing.T) {
			t.Run("delete user", func(t *testing.T) {
				usr := &os.User{
					Username: fakeUsername,
				}
				m := &MockSystem{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.State = user.StateAbsent

				m.On("Lookup", u.Username).Return(usr, nil)
				m.On("DelUser", u.Username).Return(nil)
				status, err := u.Apply()

				m.AssertCalled(t, "DelUser", u.Username)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("deleted user %s", u.Username), status.Messages()[0])
			})

			t.Run("no delete-error deleting user", func(t *testing.T) {
				usr := &os.User{
					Username: fakeUsername,
				}
				m := &MockSystem{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.State = user.StateAbsent

				m.On("Lookup", u.Username).Return(usr, nil)
				m.On("DelUser", u.Username).Return(fmt.Errorf(""))
				status, err := u.Apply()

				m.AssertCalled(t, "DelUser", u.Username)
				assert.EqualError(t, err, "user delete: ")
				assert.Equal(t, resource.StatusFatal, status.StatusCode())
				assert.Equal(t, fmt.Sprintf("error deleting user %s", u.Username), status.Messages()[0])
			})

			t.Run("no delete-will not attempt delete", func(t *testing.T) {
				usr := &os.User{
					Username: fakeUsername,
				}
				m := &MockSystem{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.State = user.StateAbsent

				m.On("Lookup", u.Username).Return(usr, os.UnknownUserError(""))
				m.On("DelUser", u.Username).Return(nil)
				status, err := u.Apply()

				m.AssertNotCalled(t, "DelUser", u.Username)
				assert.EqualError(t, err, fmt.Sprintf("will not attempt to delete user %s", u.Username))
				assert.Equal(t, resource.StatusCantChange, status.StatusCode())
			})
		})

		t.Run("uid provided", func(t *testing.T) {
			t.Run("delete user with uid", func(t *testing.T) {
				usr := &os.User{
					Username: fakeUsername,
					Uid:      fakeUID,
				}
				m := &MockSystem{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.UID = usr.Uid
				u.State = user.StateAbsent

				m.On("Lookup", u.Username).Return(usr, nil)
				m.On("LookupID", u.UID).Return(usr, nil)
				m.On("DelUser", u.Username).Return(nil)
				status, err := u.Apply()

				m.AssertCalled(t, "DelUser", u.Username)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("deleted user %s with uid %s", u.Username, u.UID), status.Messages()[0])
			})

			t.Run("no delete-error deleting user", func(t *testing.T) {
				usr := &os.User{
					Username: fakeUsername,
					Uid:      fakeUID,
				}
				m := &MockSystem{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.UID = usr.Uid
				u.State = user.StateAbsent

				m.On("Lookup", u.Username).Return(usr, nil)
				m.On("LookupID", u.UID).Return(usr, nil)
				m.On("DelUser", u.Username).Return(fmt.Errorf(""))
				status, err := u.Apply()

				m.AssertCalled(t, "DelUser", u.Username)
				assert.EqualError(t, err, "user delete: ")
				assert.Equal(t, resource.StatusFatal, status.StatusCode())
				assert.Equal(t, fmt.Sprintf("error deleting user %s with uid %s", u.Username, u.UID), status.Messages()[0])
			})

			t.Run("no delete-will not attempt delete", func(t *testing.T) {
				usr := &os.User{
					Username: fakeUsername,
					Uid:      fakeUID,
				}
				m := &MockSystem{}
				u := user.NewUser(m)
				u.Username = usr.Username
				u.UID = usr.Uid
				u.State = user.StateAbsent

				m.On("Lookup", u.Username).Return(usr, os.UnknownUserError(""))
				m.On("LookupID", u.UID).Return(usr, nil)
				m.On("DelUser", u.Username).Return(nil)
				status, err := u.Apply()

				m.AssertNotCalled(t, "DelUser", u.Username)
				assert.EqualError(t, err, fmt.Sprintf("will not attempt to delete user %s with uid %s", u.Username, u.UID))
				assert.Equal(t, resource.StatusCantChange, status.StatusCode())
			})
		})
	})

	t.Run("state unknown", func(t *testing.T) {
		usr := &os.User{
			Username: fakeUsername,
			Uid:      fakeUID,
		}
		m := &MockSystem{}
		d := &MockDiff{}
		u := user.NewUser(m)
		u.Username = usr.Username
		u.UID = usr.Uid
		u.State = "test"
		options := user.AddUserOptions{UID: u.UID}

		m.On("Lookup", u.Username).Return(usr, nil)
		m.On("LookupID", u.UID).Return(usr, nil)
		d.On("DiffAdd", u.Status).Return(options, nil)
		m.On("AddUser", u.Username, &options).Return(nil)
		m.On("DelUser", u.Username).Return(nil)
		status, err := u.Apply()

		d.AssertNotCalled(t, "DiffAdd", u)
		m.AssertNotCalled(t, "AddUser", u.Username, &options)
		m.AssertNotCalled(t, "DelUser", u.Username)
		assert.EqualError(t, err, fmt.Sprintf("user: unrecognized state %s", u.State))
		assert.Equal(t, resource.StatusFatal, status.StatusCode())
	})
}

// TestDiffAdd tests DiffAdd for user
func TestDiffAdd(t *testing.T) {
	t.Parallel()

	t.Run("set all options", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.Username = fakeUsername
		u.UID = fakeUID
		u.GroupName = existingGroupName
		u.Name = "test"
		u.HomeDir = "/tmp/test"
		u.Status = resource.NewStatus()

		expected := &user.AddUserOptions{
			UID:       u.UID,
			Group:     u.GroupName,
			Comment:   u.Name,
			Directory: u.HomeDir,
		}

		options, err := u.DiffAdd(u.Status)

		assert.NoError(t, err)
		assert.Equal(t, expected, options)
		assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
		assert.True(t, u.Status.HasChanges())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["username"].Original())
		assert.Equal(t, u.Username, u.Status.Diffs()["username"].Current())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["group"].Original())
		assert.Equal(t, u.GroupName, u.Status.Diffs()["group"].Current())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["uid"].Original())
		assert.Equal(t, u.UID, u.Status.Diffs()["uid"].Current())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["comment"].Original())
		assert.Equal(t, u.Name, u.Status.Diffs()["comment"].Current())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["home_dir"].Original())
		assert.Equal(t, u.HomeDir, u.Status.Diffs()["home_dir"].Current())
	})

	t.Run("username", func(t *testing.T) {
		t.Run("group exists-provide groupname", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = existingGroupName
			u.GroupName = existingGroupName
			u.Status = resource.NewStatus()

			expected := &user.AddUserOptions{
				Group: u.GroupName,
			}

			options, err := u.DiffAdd(u.Status)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["username"].Original())
			assert.Equal(t, u.Username, u.Status.Diffs()["username"].Current())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["group"].Original())
			assert.Equal(t, u.GroupName, u.Status.Diffs()["group"].Current())
		})

		t.Run("error-group exists", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = existingGroupName
			u.Status = resource.NewStatus()

			options, err := u.DiffAdd(u.Status)

			assert.EqualError(t, err, fmt.Sprintf("group %s exists", u.Username))
			assert.Nil(t, options)
			assert.Equal(t, resource.StatusCantChange, u.Status.StatusCode())
			assert.Equal(t, "if you want to add this user to that group, use the groupname field", u.Status.Messages()[0])
			assert.True(t, u.Status.HasChanges())
		})
	})

	t.Run("uid", func(t *testing.T) {
		t.Run("uid not found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = fakeUsername
			u.UID = fakeUID
			u.Status = resource.NewStatus()

			expected := &user.AddUserOptions{
				UID: u.UID,
			}

			options, err := u.DiffAdd(u.Status)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["username"].Original())
			assert.Equal(t, u.Username, u.Status.Diffs()["username"].Current())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["uid"].Original())
			assert.Equal(t, u.UID, u.Status.Diffs()["uid"].Current())
		})

		t.Run("error-uid found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = fakeUsername
			u.UID = currUID
			u.Status = resource.NewStatus()

			options, err := u.DiffAdd(u.Status)

			assert.EqualError(t, err, fmt.Sprintf("uid %s already exists", u.UID))
			assert.Nil(t, options)
			assert.Equal(t, resource.StatusCantChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
		})
	})

	t.Run("group", func(t *testing.T) {
		t.Run("with groupname", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = fakeUsername
			u.GroupName = existingGroupName
			u.Status = resource.NewStatus()

			expected := &user.AddUserOptions{
				Group: u.GroupName,
			}

			options, err := u.DiffAdd(u.Status)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["username"].Original())
			assert.Equal(t, u.Username, u.Status.Diffs()["username"].Current())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["group"].Original())
			assert.Equal(t, u.GroupName, u.Status.Diffs()["group"].Current())
		})

		t.Run("error-groupname not found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = fakeUsername
			u.GroupName = fakeGroupName
			u.Status = resource.NewStatus()

			options, err := u.DiffAdd(u.Status)

			assert.EqualError(t, err, fmt.Sprintf("group %s does not exist", u.GroupName))
			assert.Nil(t, options)
			assert.Equal(t, resource.StatusCantChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
		})

		t.Run("with gid", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = fakeUsername
			u.GID = existingGID
			u.Status = resource.NewStatus()

			expected := &user.AddUserOptions{
				Group: u.GID,
			}

			options, err := u.DiffAdd(u.Status)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["username"].Original())
			assert.Equal(t, u.Username, u.Status.Diffs()["username"].Current())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["gid"].Original())
			assert.Equal(t, u.GID, u.Status.Diffs()["gid"].Current())
		})

		t.Run("error-gid not found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = fakeUsername
			u.GID = fakeGID
			u.Status = resource.NewStatus()

			options, err := u.DiffAdd(u.Status)

			assert.EqualError(t, err, fmt.Sprintf("group gid %s does not exist", u.GID))
			assert.Nil(t, options)
			assert.Equal(t, resource.StatusCantChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
		})

		t.Run("user with groupname and gid", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = fakeUsername
			u.GroupName = existingGroupName
			u.GID = existingGID
			u.Status = resource.NewStatus()

			expected := &user.AddUserOptions{
				Group: u.GroupName,
			}

			options, err := u.DiffAdd(u.Status)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["username"].Original())
			assert.Equal(t, u.Username, u.Status.Diffs()["username"].Current())
			assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["group"].Original())
			assert.Equal(t, u.GroupName, u.Status.Diffs()["group"].Current())
		})
	})

	t.Run("comment", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.Username = fakeUsername
		u.Name = "test"
		u.Status = resource.NewStatus()

		expected := &user.AddUserOptions{
			Comment: u.Name,
		}

		options, err := u.DiffAdd(u.Status)

		assert.NoError(t, err)
		assert.Equal(t, expected, options)
		assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
		assert.True(t, u.Status.HasChanges())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["username"].Original())
		assert.Equal(t, u.Username, u.Status.Diffs()["username"].Current())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["comment"].Original())
		assert.Equal(t, u.Name, u.Status.Diffs()["comment"].Current())
	})

	t.Run("directory", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.Username = fakeUsername
		u.HomeDir = "/tmp/test"
		u.Status = resource.NewStatus()

		expected := &user.AddUserOptions{
			Directory: u.HomeDir,
		}

		options, err := u.DiffAdd(u.Status)

		assert.NoError(t, err)
		assert.Equal(t, expected, options)
		assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
		assert.True(t, u.Status.HasChanges())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["username"].Original())
		assert.Equal(t, u.Username, u.Status.Diffs()["username"].Current())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["home_dir"].Original())
		assert.Equal(t, u.HomeDir, u.Status.Diffs()["home_dir"].Current())
	})

	t.Run("no options", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.Username = fakeUsername
		u.Status = resource.NewStatus()

		expected := &user.AddUserOptions{}

		options, err := u.DiffAdd(u.Status)

		assert.NoError(t, err)
		assert.Equal(t, expected, options)
		assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
		assert.True(t, u.Status.HasChanges())
		assert.Equal(t, fmt.Sprintf("<%s>", string(user.StateAbsent)), u.Status.Diffs()["username"].Original())
		assert.Equal(t, u.Username, u.Status.Diffs()["username"].Current())
	})
}

// TestDiffMod tests DiffMod for user
func TestDiffMod(t *testing.T) {
	t.Parallel()

	t.Run("set all options", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.Username = currUsername
		u.NewUsername = fakeUsername
		u.UID = fakeUID
		u.GID = existingGID
		u.Name = "test"
		u.HomeDir = "/tmp/test"
		u.MoveDir = true
		u.Status = resource.NewStatus()

		expected := &user.ModUserOptions{
			Username:  u.NewUsername,
			UID:       u.UID,
			Group:     u.GID,
			Comment:   u.Name,
			Directory: u.HomeDir,
			MoveDir:   true,
		}

		options, err := u.DiffMod(u.Status, currUser)

		assert.NoError(t, err)
		assert.Equal(t, expected, options)
		assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
		assert.True(t, u.Status.HasChanges())
		assert.Equal(t, currUser.Username, u.Status.Diffs()["username"].Original())
		assert.Equal(t, u.NewUsername, u.Status.Diffs()["username"].Current())
		assert.Equal(t, currGID, u.Status.Diffs()["gid"].Original())
		assert.Equal(t, u.GID, u.Status.Diffs()["gid"].Current())
		assert.Equal(t, currUID, u.Status.Diffs()["uid"].Original())
		assert.Equal(t, u.UID, u.Status.Diffs()["uid"].Current())
		assert.Equal(t, currUser.Name, u.Status.Diffs()["comment"].Original())
		assert.Equal(t, u.Name, u.Status.Diffs()["comment"].Current())
		assert.Equal(t, currUser.HomeDir, u.Status.Diffs()["home_dir"].Original())
		assert.Equal(t, u.HomeDir, u.Status.Diffs()["home_dir"].Current())
		assert.Equal(t, currUser.HomeDir, u.Status.Diffs()["home_dir contents"].Original())
		assert.Equal(t, u.HomeDir, u.Status.Diffs()["home_dir contents"].Current())
	})

	t.Run("username", func(t *testing.T) {
		t.Run("user not found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.NewUsername = fakeUsername
			u.Status = resource.NewStatus()

			expected := &user.ModUserOptions{
				Username: u.NewUsername,
			}

			options, err := u.DiffMod(u.Status, currUser)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, currUser.Username, u.Status.Diffs()["username"].Original())
			assert.Equal(t, u.NewUsername, u.Status.Diffs()["username"].Current())
		})

		t.Run("error-user found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.NewUsername = existingUser.Username
			u.Status = resource.NewStatus()

			options, err := u.DiffMod(u.Status, currUser)

			assert.EqualError(t, err, fmt.Sprintf("user %s already exists", u.NewUsername))
			assert.Nil(t, options)
			assert.Equal(t, resource.StatusCantChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
		})
	})

	t.Run("uid", func(t *testing.T) {
		t.Run("uid not found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.UID = fakeUID
			u.Status = resource.NewStatus()

			expected := &user.ModUserOptions{
				UID: u.UID,
			}

			options, err := u.DiffMod(u.Status, currUser)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, currUID, u.Status.Diffs()["uid"].Original())
			assert.Equal(t, u.UID, u.Status.Diffs()["uid"].Current())
		})

		t.Run("error-uid found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.UID = existingUID
			u.Status = resource.NewStatus()

			options, err := u.DiffMod(u.Status, currUser)

			assert.EqualError(t, err, fmt.Sprintf("uid %s already exists", u.UID))
			assert.Nil(t, options)
			assert.Equal(t, resource.StatusCantChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
		})

		t.Run("current uid-other options", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.UID = currUID
			u.Name = "test"
			u.Status = resource.NewStatus()

			expected := &user.ModUserOptions{
				Comment: "test",
			}

			options, err := u.DiffMod(u.Status, currUser)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, currUser.Name, u.Status.Diffs()["comment"].Original())
			assert.Equal(t, u.Name, u.Status.Diffs()["comment"].Current())
			assert.Equal(t, expected.UID, "")
		})

		t.Run("current uid-no other options", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.UID = currUID
			u.Status = resource.NewStatus()

			expected := &user.ModUserOptions{}

			options, err := u.DiffMod(u.Status, currUser)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusNoChange, u.Status.StatusCode())
			assert.False(t, u.Status.HasChanges())
		})
	})

	t.Run("group", func(t *testing.T) {
		t.Run("with groupname", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.GroupName = existingGroupName
			u.Status = resource.NewStatus()

			expected := &user.ModUserOptions{
				Group: u.GroupName,
			}

			options, err := u.DiffMod(u.Status, currUser)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, currGroupName, u.Status.Diffs()["group"].Original())
			assert.Equal(t, u.GroupName, u.Status.Diffs()["group"].Current())
		})

		t.Run("error-groupname not found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.GroupName = fakeGroupName
			u.Status = resource.NewStatus()

			options, err := u.DiffMod(u.Status, currUser)

			assert.EqualError(t, err, fmt.Sprintf("group %s does not exist", u.GroupName))
			assert.Nil(t, options)
			assert.Equal(t, resource.StatusCantChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
		})

		t.Run("with gid", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.GID = existingGID
			u.Status = resource.NewStatus()

			expected := &user.ModUserOptions{
				Group: u.GID,
			}

			options, err := u.DiffMod(u.Status, currUser)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, currGID, u.Status.Diffs()["gid"].Original())
			assert.Equal(t, u.GID, u.Status.Diffs()["gid"].Current())
		})

		t.Run("error-gid not found", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.GID = fakeGID
			u.Status = resource.NewStatus()

			options, err := u.DiffMod(u.Status, currUser)

			assert.EqualError(t, err, fmt.Sprintf("group gid %s does not exist", u.GID))
			assert.Nil(t, options)
			assert.Equal(t, resource.StatusCantChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
		})

		t.Run("user with groupname and gid", func(t *testing.T) {
			u := user.NewUser(new(user.System))
			u.Username = currUsername
			u.GroupName = existingGroupName
			u.GID = existingGID
			u.Status = resource.NewStatus()

			expected := &user.ModUserOptions{
				Group: u.GroupName,
			}

			options, err := u.DiffMod(u.Status, currUser)

			assert.NoError(t, err)
			assert.Equal(t, expected, options)
			assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
			assert.True(t, u.Status.HasChanges())
			assert.Equal(t, currGroupName, u.Status.Diffs()["group"].Original())
			assert.Equal(t, u.GroupName, u.Status.Diffs()["group"].Current())
		})
	})

	t.Run("comment", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.Username = currUsername
		u.Name = "test"
		u.Status = resource.NewStatus()

		expected := &user.ModUserOptions{
			Comment: u.Name,
		}

		options, err := u.DiffMod(u.Status, currUser)

		assert.NoError(t, err)
		assert.Equal(t, expected, options)
		assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
		assert.True(t, u.Status.HasChanges())
		assert.Equal(t, currUser.Name, u.Status.Diffs()["comment"].Original())
		assert.Equal(t, u.Name, u.Status.Diffs()["comment"].Current())
	})

	t.Run("directory", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.Username = currUsername
		u.HomeDir = "/tmp/test"
		u.Status = resource.NewStatus()

		expected := &user.ModUserOptions{
			Directory: u.HomeDir,
		}

		options, err := u.DiffMod(u.Status, currUser)

		assert.NoError(t, err)
		assert.Equal(t, expected, options)
		assert.Equal(t, resource.StatusWillChange, u.Status.StatusCode())
		assert.True(t, u.Status.HasChanges())
		assert.Equal(t, currUser.HomeDir, u.Status.Diffs()["home_dir"].Original())
		assert.Equal(t, u.HomeDir, u.Status.Diffs()["home_dir"].Current())
	})

	t.Run("no options", func(t *testing.T) {
		u := user.NewUser(new(user.System))
		u.Username = currUsername
		u.Status = resource.NewStatus()

		expected := &user.ModUserOptions{}

		options, err := u.DiffMod(u.Status, currUser)

		assert.NoError(t, err)
		assert.Equal(t, expected, options)
		assert.Equal(t, resource.StatusNoChange, u.Status.StatusCode())
		assert.False(t, u.Status.HasChanges())
	})
}

// setUid is used to find a uid that exists, but is not
// a match for the current user name (currUsername).
func setUid() (string, error) {
	for i := 0; i <= maxUID; i++ {
		uid := strconv.Itoa(i)
		user, err := os.LookupId(uid)
		if err == nil && user.Username != currUsername {
			return uid, nil
		}
	}
	return "", fmt.Errorf("setUid: could not set uid")
}

// setFakeUid is used to set a uid that does not exist.
func setFakeUid() (string, error) {
	for i := minUID; i <= maxUID; i++ {
		uid := strconv.Itoa(i)
		_, err := os.LookupId(uid)
		if err != nil {
			return uid, nil
		}
	}
	return "", fmt.Errorf("setFakeUid: could not set uid")
}

// setGid is used to find a gid that exists, but is not
// the gid for the current user.
func setGid() (string, error) {
	for i := 0; i <= maxGID; i++ {
		gid := strconv.Itoa(i)
		_, err := os.LookupGroupId(gid)
		if err == nil && gid != currGID {
			return gid, nil
		}
	}
	return "", fmt.Errorf("setGid: could not set gid")
}

// setFakeGid is used to set a gid that does not exist.
func setFakeGid() (string, error) {
	for i := minGID; i <= maxGID; i++ {
		gid := strconv.Itoa(i)
		_, err := os.LookupGroupId(gid)
		if err != nil {
			return gid, nil
		}
	}
	return "", fmt.Errorf("setFakeGid: could not set gid")
}

// MockDiff is a mock implementation for user diffs
type MockDiff struct {
	mock.Mock
}

// DiffAdd sets the diffs and options for adding a user
func (m *MockDiff) DiffAdd(r resource.Status) (*user.AddUserOptions, error) {
	args := m.Called(r)
	return args.Get(0).(*user.AddUserOptions), args.Error(1)
}

// DiffMod sets the diffs and options for modifying a user
func (m *MockDiff) DiffMod(r resource.Status, u *user.User) (*user.ModUserOptions, error) {
	args := m.Called(r, u)
	return args.Get(0).(*user.ModUserOptions), args.Error(1)
}

// MockSystem is a mock implementation of user.System
type MockSystem struct {
	mock.Mock
}

// AddUser adds a user
func (m *MockSystem) AddUser(name string, options *user.AddUserOptions) error {
	args := m.Called(name, options)
	return args.Error(0)
}

// DelUser deletes a user
func (m *MockSystem) DelUser(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// ModUser modifies a user
func (m *MockSystem) ModUser(name string, options *user.ModUserOptions) error {
	args := m.Called(name, options)
	return args.Error(0)
}

// Lookup looks up a user by name
func (m *MockSystem) Lookup(name string) (*os.User, error) {
	args := m.Called(name)
	return args.Get(0).(*os.User), args.Error(1)
}

// LookupID looks up a user by ID
func (m *MockSystem) LookupID(uid string) (*os.User, error) {
	args := m.Called(uid)
	return args.Get(0).(*os.User), args.Error(1)
}

// LookupGroup looks up a group by name
func (m *MockSystem) LookupGroup(name string) (*os.Group, error) {
	args := m.Called(name)
	return args.Get(0).(*os.Group), args.Error(1)
}

// LookupGroupID looks up a group by ID
func (m *MockSystem) LookupGroupID(gid string) (*os.Group, error) {
	args := m.Called(gid)
	return args.Get(0).(*os.Group), args.Error(1)
}
