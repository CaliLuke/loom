package testdata

import . "github.com/CaliLuke/loom/dsl"


var ActivityFeedDSL = func() {
	var CommentAdded = Type("CommentAddedActivity", func() {
		Attribute("comment_id", String, func() {
			Example("cmt_17")
		})
		Attribute("thread_id", String, func() {
			Example("thr_42")
		})
		Attribute("author", String, func() {
			Example("alice")
		})
		Required("comment_id", "thread_id", "author")
	})
	var TaskClosed = Type("TaskClosedActivity", func() {
		Attribute("task_id", String, func() {
			Example("TASK-9")
		})
		Attribute("closed_by", String, func() {
			Example("bob")
		})
		Required("task_id", "closed_by")
	})
	var ActivityEnvelope = Type("ActivityEnvelope", func() {
		OneOf("entry", func() {
			Attribute("comment_added", CommentAdded, func() {
				Meta("oneof:type:tag", "comment_added")
			})
			Attribute("task_closed", TaskClosed, func() {
				Meta("oneof:type:tag", "task_closed")
			})
		})
		Meta("oneof:type:field", "kind")
		Meta("oneof:value:field", "payload")
	})
	var ValidationError = Type("ActivityValidationError", func() {
		Attribute("message", String, func() {
			Example("cursor must be a base64 token")
		})
		Attribute("field", String, func() {
			Example("cursor")
		})
		Required("message", "field")
	})
	var _ = API("activity-feed", func() {
		Title("Activity Feed API")
		Description("Union-heavy audit and activity retrieval endpoints.")
		Meta("openapi:closed-objects", "true")
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("activity-feed", func() {
			Host("api", func() {
				URI("https://api.activity-feed.example.test")
			})
		})
	})
	Service("activityFeed", func() {
		Description("Read activity entries across project workstreams.")

		Method("listActivities", func() {
			NoSecurity()
			Payload(func() {
				Attribute("project_id", String, func() {
					Example("proj_9")
				})
				Attribute("cursor", String, func() {
					Example("cursor_2")
				})
				Required("project_id")
			})
			Result(ArrayOf(ActivityEnvelope))
			Error("bad_request", ValidationError)
			HTTP(func() {
				GET("/projects/{project_id}/activities")
				Param("project_id")
				Param("cursor")
				Response(StatusOK)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("showActivity", func() {
			NoSecurity()
			Payload(func() {
				Attribute("project_id", String, func() {
					Example("proj_9")
				})
				Attribute("activity_id", String, func() {
					Example("act_4")
				})
				Required("project_id", "activity_id")
			})
			Result(ActivityEnvelope)
			Error("bad_request", ValidationError)
			HTTP(func() {
				GET("/projects/{project_id}/activities/{activity_id}")
				Param("project_id")
				Param("activity_id")
				Response(StatusOK)
				Response("bad_request", StatusBadRequest)
			})
		})
	})
}


