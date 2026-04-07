package testdata

import . "github.com/CaliLuke/loom/dsl"


var MealPlannerDSL = func() {
	var accessToken = JWTSecurity("access_token", func() {
		Description("Bearer token used by web and mobile clients.")
	})
	var browserSession = APIKeySecurity("browser_session", func() {
		Description("Browser session cookie used by the meal planner web app.")
	})
	var appSession = SessionAuth("meal_planner_session", func() {
		BearerTransport(accessToken, "auth")
		CookieTransport(browserSession, "browser_session", func() {
			CookieName("__Host-mealplanner_session")
		})
	})

	var Recipe = ResultType("application/vnd.mealplanner.recipe", func() {
		TypeName("Recipe")
		Description("A saved recipe with planning metadata.")
		Attributes(func() {
			Attribute("id", String, func() {
				Example("recipe_42")
			})
			Attribute("title", String, func() {
				Example("Sheet Pan Gnocchi")
			})
			Attribute("minutes", Int, func() {
				Minimum(5)
				Example(30)
			})
			Attribute("difficulty", String, func() {
				Enum("easy", "medium", "hard")
				Example("easy")
			})
			Attribute("tags", ArrayOf(String), func() {
				Example([]string{"weeknight", "vegetarian"})
			})
			Attribute("ingredients", ArrayOf(String), func() {
				Example([]string{"gnocchi", "tomatoes", "spinach"})
			})
			Required("id", "title", "minutes", "difficulty")
		})
		View("default", func() {
			Attribute("id")
			Attribute("title")
			Attribute("minutes")
			Attribute("difficulty")
			Attribute("tags")
			Attribute("ingredients")
		})
		View("summary", func() {
			Attribute("id")
			Attribute("title")
			Attribute("minutes")
			Attribute("difficulty")
		})
	})

	var MealPlanEntry = Type("MealPlanEntry", func() {
		Attribute("day", String, func() {
			Enum("monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday")
			Example("monday")
		})
		Attribute("slot", String, func() {
			Enum("breakfast", "lunch", "dinner")
			Example("dinner")
		})
		Attribute("recipe_id", String, func() {
			Example("recipe_42")
		})
		Required("day", "slot", "recipe_id")
	})

	var MealPlan = Type("MealPlan", func() {
		Description("A meal plan covering one planning week.")
		Attribute("week_start", String, func() {
			Format(FormatDate)
			Example("2026-03-16")
		})
		Attribute("meals", ArrayOf(MealPlanEntry))
		Required("week_start")
	})

	var MealPlanPatch = Type("MealPlanPatch", func() {
		Description("A partial meal plan update.")
		Attribute("meals", ArrayOf(MealPlanEntry))
	})

	var PantryImportReceipt = Type("PantryImportReceipt", func() {
		Attribute("import_id", String, func() {
			Example("import_7")
		})
		Attribute("accepted_items", Int, func() {
			Example(12)
		})
		Required("import_id", "accepted_items")
	})

	var SingleRecipeSelection = Type("SingleRecipeSelection", func() {
		Attribute("recipe_id", String, func() {
			Example("recipe_42")
		})
		Required("recipe_id")
	})

	var RecipePairSelection = Type("RecipePairSelection", func() {
		Attribute("primary_recipe_id", String, func() {
			Example("recipe_42")
		})
		Attribute("backup_recipe_id", String, func() {
			Example("recipe_84")
		})
		Required("primary_recipe_id", "backup_recipe_id")
	})

	var RecipeSelection = Type("RecipeSelection", func() {
		OneOf("selection", func() {
			Meta("oneof:type:field", "mode")
			Meta("oneof:value:field", "value")
			Attribute("single", SingleRecipeSelection)
			Attribute("pair", RecipePairSelection)
		})
	})

	API("mealplanner", func() {
		Meta("openapi:closed-objects", "true")
		Title("Meal Planner API")
		Description("Plan weekly meals, manage recipes, and coordinate pantry imports.")
		Version("2026-03-19")
		TermsOfService("https://mealplanner.example.test/terms")
		Contact(func() {
			Name("Meal Planner Support")
			Email("support@mealplanner.example.test")
			URL("https://mealplanner.example.test/support")
		})
		License(func() {
			Name("Apache-2.0")
			URL("https://www.apache.org/licenses/LICENSE-2.0.html")
		})
		Server("mealplanner", func() {
			Host("production", func() {
				URI("https://api.mealplanner.example.test")
			})
		})
	})

	Service("mealPlanner", func() {
		Description("Plan meals, browse recipes, and import pantry inventory.")
		SessionSecurity(appSession)

		Method("health", func() {
			NoSecurity()
			Error("rate_limited")
			Result(func() {
				Attribute("status", String, func() {
					Enum("ok")
					Example("ok")
				})
				Attribute("checked_at", String, func() {
					Format(FormatDateTime)
					Example("2026-03-19T12:00:00Z")
				})
				Required("status", "checked_at")
			})
			HTTP(func() {
				GET("/healthz")
				Response(StatusOK)
				Response("rate_limited", StatusTooManyRequests)
			})
		})

		Method("listRecipes", func() {
			Error("bad_request")
			Result(CollectionOf(Recipe), func() {
				View("summary")
			})
			Payload(func() {
				Attribute("tag", String, func() {
					Example("vegetarian")
				})
				Attribute("cursor", String, func() {
					Example("cursor_2")
				})
				Attribute("limit", Int, func() {
					Minimum(1)
					Maximum(100)
					Example(20)
				})
			})
			HTTP(func() {
				GET("/recipes")
				Param("tag")
				Param("cursor")
				Param("limit")
				AuthErrorResponses()
				Response(StatusOK)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("showRecipe", func() {
			Error("not_found")
			Result(Recipe)
			Payload(func() {
				Attribute("recipe_id", String, func() {
					Pattern("^recipe_[0-9]+$")
					Example("recipe_42")
				})
				Required("recipe_id")
			})
			HTTP(func() {
				GET("/recipes/{recipe_id}")
				Param("recipe_id")
				AuthErrorResponses()
				Response(StatusOK)
				Response("not_found", StatusNotFound)
			})
		})

		Method("upsertMealPlan", func() {
			Error("bad_request")
			Result(MealPlan)
			Payload(func() {
				Attribute("week_start", String, func() {
					Format(FormatDate)
					Example("2026-03-16")
				})
				Attribute("body", MealPlanPatch)
				Required("week_start")
			})
			HTTP(func() {
				PATCH("/meal-plans/{week_start}")
				Param("week_start")
				Body("body")
				OptionalRequestBody()
				AuthErrorResponses()
				Response(StatusOK)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("shareMealPlan", func() {
			Error("bad_request")
			Result(Empty)
			Payload(func() {
				Attribute("week_start", String, func() {
					Format(FormatDate)
					Example("2026-03-16")
				})
				Attribute("email", String, func() {
					Format(FormatEmail)
					Example("friend@example.test")
				})
				Attribute("note", String, func() {
					MaxLength(280)
					Example("Dinner plan for next week.")
				})
				Attribute("include_shopping_list", Boolean, func() {
					Example(true)
				})
				Required("week_start", "email")
			})
			HTTP(func() {
				POST("/meal-plans/{week_start}/share")
				Param("week_start")
				FormRequest()
				AuthErrorResponses()
				Response(StatusNoContent)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("previewSelection", func() {
			Error("bad_request")
			Result(Empty)
			Payload(func() {
				Attribute("selection", RecipeSelection, func() {
					Example(map[string]any{
						"mode": "single",
						"value": map[string]any{
							"recipe_id": "recipe_42",
						},
					})
				})
				Required("selection")
			})
			HTTP(func() {
				POST("/plans/preview")
				AuthErrorResponses()
				Response(StatusNoContent)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("previewSelectionSuppressed", func() {
			Error("bad_request")
			Result(Empty)
			Payload(func() {
				Attribute("selection", RecipeSelection, func() {
					Meta("openapi:example", "false")
				})
				Required("selection")
			})
			HTTP(func() {
				POST("/plans/preview-suppressed")
				AuthErrorResponses()
				Response(StatusNoContent)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("importPantry", func() {
			Error("bad_request")
			Result(PantryImportReceipt)
			Payload(func() {
				Attribute("pantry_id", String, func() {
					Example("pantry_12")
				})
				Attribute("file", Bytes)
				Attribute("filename", String)
				Attribute("content_type", String)
				Attribute("label", String, func() {
					Pattern("^scan-[a-z]+$")
					Example("scan-weekly")
				})
				Required("pantry_id", "file", "label")
			})
			HTTP(func() {
				POST("/pantries/{pantry_id}/imports")
				Param("pantry_id")
				MultipartRequest()
				AuthErrorResponses()
				Response(StatusAccepted)
				Response("bad_request", StatusBadRequest)
			})
		})

		Method("watchPlannerEvents", func() {
			NoSecurity()
			Error("rate_limited")
			StreamingResult(func() {
				Attribute("event", String, func() {
					Example("plan.updated")
				})
				Attribute("data", func() {
					Attribute("meal_plan_id", String, func() {
						Example("plan_12")
					})
					Required("meal_plan_id")
				})
				Required("event", "data")
			})
			HTTP(func() {
				GET("/events")
				ServerSentEvents()
				Response("rate_limited", StatusTooManyRequests)
			})
		})
	})
}


