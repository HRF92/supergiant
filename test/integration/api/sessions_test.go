package api

import (
	"testing"

	"github.com/supergiant/supergiant/pkg/core"
	"github.com/supergiant/supergiant/pkg/model"

	. "github.com/smartystreets/goconvey/convey"
)

func createSession(c *core.Core, username string, password string) *model.Session {
	session := &model.Session{
		User: &model.User{
			Username: username,
			Password: password,
		},
	}
	c.Sessions.Create(session)
	return session
}

func createUserSession(c *core.Core) *model.Session {
	return createSession(c, "user", "password")
}

func createAdminSession(c *core.Core) *model.Session {
	return createSession(c, "bossman", "password")
}

//------------------------------------------------------------------------------

func TestSessionsList(t *testing.T) {
	srv := newTestServer()
	go srv.Start()
	defer srv.Stop()

	user, admin := createUserAndAdmin(srv.Core)
	createUserSession(srv.Core)
	createAdminSession(srv.Core)

	Convey("Given a user and an admin, both logged-in", t, func() {

		Convey("When the user Lists Sessions", func() {
			sg := srv.Core.NewAPIClient("token", user.APIToken)
			var sessions []*model.Session
			err := sg.Sessions.List(&sessions)

			Convey("They should see only their own Session", func() {
				So(err, ShouldBeNil)
				So(*sessions[0].UserID, ShouldEqual, *user.ID)
			})
		})

		Convey("When the admin Lists Sessions", func() {
			sg := srv.Core.NewAPIClient("token", admin.APIToken)
			var sessions []*model.Session
			err := sg.Sessions.List(&sessions)

			Convey("They should see all Sessions", func() {
				So(err, ShouldBeNil)
				So(len(sessions), ShouldEqual, 2)
			})
		})
	})
}

func TestSessionsCreate(t *testing.T) {
	srv := newTestServer()
	go srv.Start()
	defer srv.Stop()

	createUser(srv.Core)
	sg := srv.Core.NewAPIClient("", "")

	Convey("Given a User and an unauthenticated Client", t, func() {

		Convey("When a Session is Created with invalid username", func() {
			session := &model.Session{
				User: &model.User{
					Username: "userb",
					Password: "password",
				},
			}
			err := sg.Sessions.Create(session).(*model.Error)

			Convey("There should be a non-specific credential mismatch error", func() {
				So(err.Status, ShouldEqual, 400)
				So(err.Message, ShouldEqual, "Invalid credentials")
			})
		})

		Convey("When a Session is Created with invalid password", func() {
			session := &model.Session{
				User: &model.User{
					Username: "user",
					Password: "passwordb",
				},
			}
			err := sg.Sessions.Create(session).(*model.Error)

			Convey("There should be a non-specific credential mismatch error", func() {
				So(err.Status, ShouldEqual, 400)
				So(err.Message, ShouldEqual, "Invalid credentials")
			})
		})

		Convey("When a Session is Created with valid credentials", func() {
			session := &model.Session{
				User: &model.User{
					Username: "user",
					Password: "password",
				},
			}
			err := sg.Sessions.Create(session)

			Convey("There should be no error, and the Session ID should allow for API authentication", func() {
				So(err, ShouldBeNil)

				userSG := srv.Core.NewAPIClient("session", session.ID)
				apps := make([]*model.App, 0)
				authErr := userSG.Apps.List(&apps)
				So(authErr, ShouldBeNil)
			})
		})
	})
}

func TestSessionsGet(t *testing.T) {
	srv := newTestServer()
	go srv.Start()
	defer srv.Stop()

	user, admin := createUserAndAdmin(srv.Core)
	userSession := createUserSession(srv.Core)
	adminSession := createAdminSession(srv.Core)

	Convey("Given a user and an admin, both logged-in", t, func() {

		Convey("When the user Gets another User's Session", func() {
			sg := srv.Core.NewAPIClient("token", user.APIToken)
			err := sg.Sessions.Get(adminSession.ID, new(model.Session))

			Convey("They should receive a 403 Forbidden error", func() {
				So(err.(*model.Error).Status, ShouldEqual, 403)
			})
		})

		Convey("When the user Gets their own Session", func() {
			sg := srv.Core.NewAPIClient("token", user.APIToken)
			err := sg.Sessions.Get(userSession.ID, new(model.Session))

			Convey("There should be no error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When the admin Gets another User's Session", func() {
			sg := srv.Core.NewAPIClient("token", admin.APIToken)
			err := sg.Sessions.Get(userSession.ID, new(model.Session))

			Convey("There should be no error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When the admin Gets their own Session", func() {
			sg := srv.Core.NewAPIClient("token", admin.APIToken)
			err := sg.Sessions.Get(userSession.ID, new(model.Session))

			Convey("There should be no error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestSessionsDelete(t *testing.T) {
	uselessList := make([]*model.App, 0)

	Convey("Given a user and an admin, both logged-in", t, func() {
		srv := newTestServer()
		go srv.Start()
		defer srv.Stop()

		user, admin := createUserAndAdmin(srv.Core)
		userSession := createUserSession(srv.Core)
		adminSession := createAdminSession(srv.Core)

		Convey("When the user Deletes another User's Session", func() {
			sg := srv.Core.NewAPIClient("token", user.APIToken)
			err := sg.Sessions.Delete(adminSession.ID, new(model.Session))

			Convey("They should receive a 403 Forbidden error", func() {
				So(err.(*model.Error).Status, ShouldEqual, 403)
			})
		})

		Convey("When the user Deletes their own Session", func() {
			sg := srv.Core.NewAPIClient("session", userSession.ID)
			err := sg.Sessions.Delete(userSession.ID, new(model.Session))

			Convey("The Session should be deleted, and no longer allow login", func() {
				So(err, ShouldBeNil)

				authErr := srv.Core.NewAPIClient("session", userSession.ID).Apps.List(&uselessList)
				So(authErr.(*model.Error).Status, ShouldEqual, 401)
			})
		})

		Convey("When the admin Deletes another User's Session", func() {
			sg := srv.Core.NewAPIClient("token", admin.APIToken)
			err := sg.Sessions.Delete(userSession.ID, new(model.Session))

			Convey("The Session should be deleted, and no longer allow login", func() {
				So(err, ShouldBeNil)

				authErr := srv.Core.NewAPIClient("session", userSession.ID).Apps.List(&uselessList)
				So(authErr.(*model.Error).Status, ShouldEqual, 401)
			})
		})

		Convey("When the admin Deletes their own Session", func() {
			sg := srv.Core.NewAPIClient("token", admin.APIToken)
			err := sg.Sessions.Delete(adminSession.ID, new(model.Session))

			Convey("The Session should be deleted, and no longer allow login", func() {
				So(err, ShouldBeNil)

				authErr := srv.Core.NewAPIClient("session", adminSession.ID).Apps.List(&uselessList)
				So(authErr.(*model.Error).Status, ShouldEqual, 401)
			})
		})
	})
}
