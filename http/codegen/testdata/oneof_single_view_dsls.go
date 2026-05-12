package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// OneOfResultSingleViewDSL defines a ResultType whose default view drops the
// OneOf field. Client response code must therefore treat the transport body as
// the projected view, not the raw ResultType.
var OneOfResultSingleViewDSL = func() {
	animal := oneOfAnimalResultType("application/vnd.oneof-http-single-view.animal")

	Service("ServiceOneOfSingleView", func() {
		Method("MethodShowAnimal", func() {
			Payload(func() {
				Attribute("id", String)
				Required("id")
			})
			Result(animal)
			HTTP(func() {
				GET("/animals/{id}")
				Response(StatusOK)
			})
		})
	})
}

// OneOfResultCollectionSingleViewDSL defines a collection result whose
// transport body is fixed to a single view. The generated client collection
// code must project before collecting unions and body types.
var OneOfResultCollectionSingleViewDSL = func() {
	animal := oneOfAnimalResultType("application/vnd.oneof-http-collection-view.animal")

	Service("ServiceOneOfCollectionSingleView", func() {
		Method("MethodListAnimals", func() {
			Result(CollectionOf(animal), func() {
				View("default")
			})
			HTTP(func() {
				GET("/animals")
				Response(StatusOK)
			})
		})
	})
}

func oneOfAnimalResultType(mediaType string) *expr.ResultTypeExpr {
	catDetails := Type("CatDetails", func() {
		Attribute("favorite_spot", String)
		Attribute("lives_left", Int)
		Required("favorite_spot", "lives_left")
	})
	dogDetails := Type("DogDetails", func() {
		Attribute("favorite_park", String)
		Attribute("plays_fetch", Boolean)
		Required("favorite_park", "plays_fetch")
	})
	birdDetails := Type("BirdDetails", func() {
		Attribute("can_fly", Boolean)
		Attribute("vocabulary_size", Int)
		Required("can_fly", "vocabulary_size")
	})
	fishDetails := Type("FishDetails", func() {
		Attribute("water_type", String)
		Required("water_type")
	})

	return ResultType(mediaType, func() {
		TypeName("Animal")
		Attributes(func() {
			Attribute("name", String)
			OneOf("details", func() {
				Attribute("cat", catDetails)
				Attribute("dog", dogDetails)
				Attribute("bird", birdDetails)
				Attribute("fish", fishDetails)
			})
			Attribute("id", String)
		})
		View("default", func() {
			Attribute("name")
			Attribute("id")
		})
		Required("name", "id")
	})
}
