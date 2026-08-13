package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

var CookieObjectResultDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("cookie")
				})
			})
		})
	})
}

var CookieStringResultDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(String)
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("cookie")
				})
			})
		})
	})
}

const CookieMaxAgeValue = 3600

var CookieMaxAgeDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("cookie")
					CookieMaxAge(CookieMaxAgeValue)
				})
			})
		})
	})
}

const CookieDomainValue = "loom.design"

var CookieDomainDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("cookie")
					CookieDomain(CookieDomainValue)
				})
			})
		})
	})
}

const CookiePathValue = "/path"

var CookiePathDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("cookie")
					CookiePath(CookiePathValue)
				})
			})
		})
	})
}

var CookieSecureDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("cookie")
					CookieSecure()
				})
			})
		})
	})
}

var CookieInsecureDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					SessionCookie("cookie")
					CookieInsecure()
				})
			})
		})
	})
}

var CookieSecureAfterInsecureDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("cookie")
					CookieInsecure()
					CookieSecure()
				})
			})
		})
	})
}

var CookieHTTPOnlyDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("cookie")
					CookieHTTPOnly()
				})
			})
		})
	})
}

const CookieSameSiteValue = CookieSameSiteStrict

var CookieSameSiteDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("cookie")
					CookieSameSite(CookieSameSiteValue)
				})
			})
		})
	})
}

var SessionCookieDefaultsDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					SessionCookie("cookie")
				})
			})
		})
	})
}

const SessionCookieOverridePathValue = "/session"
const SessionCookieOverrideSameSiteValue = CookieSameSiteStrict

var SessionCookieOverrideDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					SessionCookie("cookie")
					CookiePath(SessionCookieOverridePathValue)
					CookieSameSite(SessionCookieOverrideSameSiteValue)
				})
			})
		})
	})
}

const SessionCookieOverrideDomainValue = "session.loom.design"
const SessionCookieOverrideMaxAgeValue = 7200

var SessionCookieOverrideAllDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					SessionCookie("cookie")
					CookiePath(SessionCookieOverridePathValue)
					CookieDomain(SessionCookieOverrideDomainValue)
					CookieMaxAge(SessionCookieOverrideMaxAgeValue)
					CookieSameSite(SessionCookieOverrideSameSiteValue)
				})
			})
		})
	})
}

var SessionCookieRepeatedSetterDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					SessionCookie("cookie")
					CookiePath("/ignored")
					CookiePath(SessionCookieOverridePathValue)
					CookieSameSite(CookieSameSiteNone)
					CookieSameSite(SessionCookieOverrideSameSiteValue)
				})
			})
		})
	})
}

var MultipleResponseCookiesDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("session", String)
				Attribute("refresh", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					SessionCookie("session:__Host-ak_session")
					Cookie("refresh:ak_refresh")
					CookiePath("/tokens")
					CookieDomain("accounts.loom.design")
				})
			})
		})
	})
}

var DuplicateResponseCookieNameDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("session", String)
				Attribute("refresh", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					Cookie("session:ak")
					Cookie("refresh:ak")
				})
			})
		})
	})
}

var InvalidCookieSetterPlacementDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					CookiePath("/session")
					Cookie("cookie")
				})
			})
		})
	})
}

var InvalidCookieInsecurePlacementDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			HTTP(func() {
				POST("/")
				Response(StatusOK, func() {
					CookieInsecure()
					Cookie("cookie")
				})
			})
		})
	})
}

var InvalidSessionCookiePlacementDSL = func() {
	Service("CookieSvc", func() {
		Method("Method", func() {
			Result(func() {
				Attribute("cookie", String)
			})
			SessionCookie("cookie")
			HTTP(func() {
				POST("/")
				Response(StatusOK)
			})
		})
	})
}
