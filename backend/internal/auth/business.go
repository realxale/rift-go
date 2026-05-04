package auth

func buisnessReg(req RegRequest) error {
	hash, err := hashPassword(req.Password)
	if err != nil {
		return err
	}
	err = RegUserDB(req.Username, hash)
	if err != nil {
		return err
	}
	return err
}
func buisnessAuth(req *RegRequest) (error, bool, string) {
	err, res := AuthUserDB(req.Username, req.Password)
	if err != nil {
		return err, false, ""
	}
	if !res {
		return nil, false, ""
	}
	jwt_token, err := GenerateJWT(req.Username)
	if err != nil {
		return err, false, ""
	}
	return nil, true, jwt_token
}

// func authJWT(req JAR)(bool,error){
// claims, err := ParseJWT(tokenString)
// if err != nil {
// 	log.Println("error:", err)
// 	return false,err
// }
// return true,nil
// }
