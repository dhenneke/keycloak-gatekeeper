/*
Copyright 2015 All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/gambol99/go-oidc/jose"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func getFakeAccessToken(t *testing.T) jose.JWT {
	testToken, err := jose.NewJWT(
		jose.JOSEHeader{
			"alg": "RS256",
		},
		jose.Claims{
			"jti": "4ee75b8e-3ee6-4382-92d4-3390b4b4937b",
			//"exp": "1450372969",
			"nbf":            0,
			"iat":            "1450372669",
			"iss":            "https://keycloak.example.com/auth/realms/commons",
			"aud":            "test",
			"sub":            "1e11e539-8256-4b3b-bda8-cc0d56cddb48",
			"typ":            "Bearer",
			"azp":            "clientid",
			"session_state":  "98f4c3d2-1b8c-4932-b8c4-92ec0ea7e195",
			"client_session": "f0105893-369a-46bc-9661-ad8c747b1a69",
			"resource_access": map[string]interface{}{
				"openvpn": map[string]interface{}{
					"roles": []string{
						"dev-vpn",
					},
				},
			},
			"email":              "gambol99@gmail.com",
			"name":               "Rohith Jayawardene",
			"family_name":        "Jayawardene",
			"preferred_username": "rjayawardene",
			"given_name":         "Rohith",
		},
	)
	if err != nil {
		t.Fatalf("unable to generate a token: %s", err)
	}

	return testToken
}

func TestGetUserContext(t *testing.T) {
	proxy := newFakeKeycloakProxy(t)
	token := getFakeAccessToken(t)

	context, err := proxy.getUserContext(token)
	assert.NoError(t, err)
	assert.NotNil(t, context)
	assert.Equal(t, "1e11e539-8256-4b3b-bda8-cc0d56cddb48", context.id)
	assert.Equal(t, "gambol99@gmail.com", context.email)
	assert.Equal(t, "rjayawardene", context.preferredName)
	roles := []string{"openvpn:dev-vpn"}
	if !reflect.DeepEqual(context.roles, roles) {
		t.Errorf("the claims are not the same, %v <-> %v", context.roles, roles)
	}
}

func TestGetSessionToken(t *testing.T) {
	proxy := newFakeKeycloakProxy(t)
	token := getFakeAccessToken(t)
	encoded := token.Encode()

	testCases := []struct {
		Context *gin.Context
		Ok      bool
	}{
		{
			Context: &gin.Context{
				Request: &http.Request{
					Header: http.Header{
						"Authorization": []string{fmt.Sprintf("Bearer %s", encoded)},
					},
				},
			},
			Ok: true,
		},
		{
			Context: &gin.Context{
				Request: &http.Request{
					Header: http.Header{},
				},
			},
		},
		// @TODO need to ather checks
	}

	for i, c := range testCases {
		token, _, err := proxy.getSessionToken(c.Context)
		if err != nil && c.Ok {
			t.Errorf("test case %d should not have errored", i)
			continue
		}
		if err != nil && !c.Ok {
			continue
		}
		if token.Encode() != encoded {
			t.Errorf("test case %d the tokens are not the same", i)
		}
	}
}

func TestEncodeState(t *testing.T) {
	proxy := newFakeKeycloakProxy(t)

	state := &sessionState{
		refreshToken: "this is a fake session",
		expireOn:     time.Now(),
	}

	session, err := proxy.encodeState(state)
	assert.NotEmpty(t, session)
	assert.NoError(t, err)
}

func TestDecodeState(t *testing.T) {
	proxy := newFakeKeycloakProxy(t)

	fakeToken := "this is a fake session"
	fakeExpiresOn := time.Now()

	state := &sessionState{
		refreshToken: fakeToken,
		expireOn:     fakeExpiresOn,
	}

	session, err := proxy.encodeState(state)
	assert.NotEmpty(t, session)
	if err != nil {
		t.Errorf("the encodeState() should not have handed an error")
		t.FailNow()
	}

	decoded, err := proxy.decodeState(session)
	assert.NotNil(t, decoded, "the session should not have been nil")
	if assert.NoError(t, err, "the decodeState() should not have thrown an error") {
		assert.Equal(t, fakeToken, decoded.refreshToken, "the token should been the same")
	}
}
