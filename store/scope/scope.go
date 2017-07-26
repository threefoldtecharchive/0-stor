package scope

import (
	"strings"
	"fmt"
	"github.com/pkg/errors"
)

type Scope struct {
	Namespace string
	Actor string
	Action string
	Organization string
	Permission string
}

func (s *Scope) Validate() error{
	if s.Action == "" ||
		s.Actor == ""||
		s.Organization == ""||
		s.Permission == ""{
		return errors.New("one or more required fields is empty")
	}

	if s.Permission != "read" &&
		s.Permission != "write" &&
		s.Permission != "delete" &&
		s.Permission != "admin"{
		return errors.New("Invalid permission")
	}

	return nil
}

func (s *Scope) Encode() (string, error){

	if err := s.Validate(); err != nil{
		return "", err
	}


	r := fmt.Sprintf("%s:%s:%s", s.Actor, s.Action, s.Organization)

	if s.Namespace == ""{
		r = fmt.Sprintf("%s.%s", r, "*")
	}else{
		r = fmt.Sprintf("%s.%s", r, s.Namespace)
	}

	if s.Permission != "admin"{
		r = fmt.Sprintf("%s.%s", r, s.Permission)
	}

	return r, nil
}

func (s *Scope) Decode(scope string) error{
	scope = strings.ToLower(scope)

	if strings.Count(scope, ":") != 2{
		return errors.New("Invalid scope string")
	}

	splitted := strings.Split(scope, ":")

	actor := splitted[0]
	action := splitted[1]

	count := strings.Count(splitted[2], ".")

	if count == 0 || count > 2{
		return errors.New("Invalid scope string")
	}

	splitted = strings.Split(splitted[2], ".")

	s.Organization = splitted[0]
	ns := splitted[1]

	// If we need to match any namespace, we leave namespace in Scope empty
	if ns == "*"{
		s.Namespace = ""
	}else{
		s.Namespace = ns
	}

	if len(splitted) == 2{
		s.Permission = "admin"
	}else{
		s.Permission = splitted[2]
	}

	s.Action = action
	s.Actor = actor

	if err := s.Validate(); err != nil{
		return err
	}

	return nil
}


