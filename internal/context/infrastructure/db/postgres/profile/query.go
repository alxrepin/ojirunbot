package profile

const upsertQuery = `
	INSERT INTO user_profiles (
		user_id, sex, age, height_cm, weight_kg, activity_level, goal,
		daily_calories_kcal, daily_protein_g, daily_fat_g, daily_carbs_g,
		formula_version
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (user_id) DO UPDATE
	SET sex=EXCLUDED.sex,
	    age=EXCLUDED.age,
	    height_cm=EXCLUDED.height_cm,
	    weight_kg=EXCLUDED.weight_kg,
	    activity_level=EXCLUDED.activity_level,
	    goal=EXCLUDED.goal,
	    daily_calories_kcal=EXCLUDED.daily_calories_kcal,
	    daily_protein_g=EXCLUDED.daily_protein_g,
	    daily_fat_g=EXCLUDED.daily_fat_g,
	    daily_carbs_g=EXCLUDED.daily_carbs_g,
	    formula_version=EXCLUDED.formula_version,
	    updated_at=now()`

const selectByUserIDQuery = `
	SELECT user_id::text, sex, age, height_cm, weight_kg, activity_level, goal,
	       daily_calories_kcal, daily_protein_g, daily_fat_g, daily_carbs_g, formula_version
	FROM user_profiles
	WHERE user_id=$1`
