package database

import (
	"fmt"
	"log"
	"student-backend/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	log.Println("🔄 Starting database migration...")

	// Сначала удаляем все таблицы в правильном порядке
	log.Println("🗑️ Dropping existing tables...")
	dropOrder := []string{
		"users",
		"students",
		"teachers",
		"groups",
	}

	for _, table := range dropOrder {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)).Error; err != nil {
			log.Printf("⚠️ Warning: Could not drop table %s: %v", table, err)
		}
	}

	// Создаем таблицы с использованием GORM AutoMigrate
	// В правильном порядке: сначала независимые таблицы, потом зависимые
	tables := []interface{}{
		&models.Group{},
		&models.Student{},
		&models.Teacher{},
		&models.User{},
	}

	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			log.Printf("❌ Error migrating table %T: %v", table, err)
			return err
		}
		log.Printf("✅ Created/Updated table for: %T", table)
	}

	// Создаем индексы вручную (если нужно)
	createIndexes(db)

	// Заполняем начальными данными
	if err := seedInitialData(db); err != nil {
		log.Printf("⚠️ Error seeding initial data: %v", err)
	}

	log.Println("✅ Database migration completed successfully!")
	return nil
}

func createIndexes(db *gorm.DB) {
	log.Println("📊 Creating indexes...")

	// Индексы для таблицы students
	db.Exec("CREATE INDEX IF NOT EXISTS idx_students_name ON students(name)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_students_surname ON students(surname)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_students_group_id ON students(group_id)")

	// Индексы для таблицы users
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)")

	// Индексы для таблицы teachers
	db.Exec("CREATE INDEX IF NOT EXISTS idx_teachers_email ON teachers(email)")

	log.Println("✅ Indexes created successfully!")
}

func seedInitialData(db *gorm.DB) error {
	log.Println("🌱 Seeding initial data...")

	// Проверяем, есть ли уже данные
	var userCount int64
	db.Model(&models.User{}).Count(&userCount)

	if userCount > 0 {
		log.Println("✅ Database already has data, skipping seed")
		return nil
	}

	// Создаем группы
	groups := []models.Group{
		{Name: "Информатика 101", Code: "INF-101"},
		{Name: "Математика 201", Code: "MATH-201"},
		{Name: "Физика 301", Code: "PHYS-301"},
	}

	for i := range groups {
		if err := db.Create(&groups[i]).Error; err != nil {
			log.Printf("❌ Error creating group: %v", err)
			return err
		}
	}

	// Хешируем пароль для админа
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Создаем администратора
	admin := models.User{
		Email:    "admin@example.com",
		Password: string(hashedPassword),
		Role:     models.RoleAdmin,
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Printf("❌ Error creating admin user: %v", err)
		return err
	}

	log.Printf("✅ Created admin user: %s (password: admin123)", admin.Email)

	// Создаем тестового студента (с пользователем)
	studentPassword, _ := bcrypt.GenerateFromPassword([]byte("student123"), bcrypt.DefaultCost)
	studentUser := models.User{
		Email:    "student@example.com",
		Password: string(studentPassword),
		Role:     models.RoleStudent,
	}

	if err := db.Create(&studentUser).Error; err != nil {
		log.Printf("❌ Error creating student user: %v", err)
	}

	student := models.Student{
		Name:    "Иван",
		Surname: "Иванов",
		Email:   "student@example.com",
		GroupID: &groups[0].ID,
		UserID:  &studentUser.ID,
	}

	if err := db.Create(&student).Error; err != nil {
		log.Printf("❌ Error creating student: %v", err)
	}

	// Обновляем связь
	db.Model(&studentUser).Update("student_id", student.ID)

	// Создаем тестового преподавателя (с пользователем)
	teacherPassword, _ := bcrypt.GenerateFromPassword([]byte("teacher123"), bcrypt.DefaultCost)
	teacherUser := models.User{
		Email:    "teacher@example.com",
		Password: string(teacherPassword),
		Role:     models.RoleTeacher,
	}

	if err := db.Create(&teacherUser).Error; err != nil {
		log.Printf("❌ Error creating teacher user: %v", err)
	}

	teacher := models.Teacher{
		Name:    "Петр",
		Surname: "Петров",
		Email:   "teacher@example.com",
		UserID:  &teacherUser.ID,
	}

	if err := db.Create(&teacher).Error; err != nil {
		log.Printf("❌ Error creating teacher: %v", err)
	}

	// Обновляем связь
	db.Model(&teacherUser).Update("teacher_id", teacher.ID)

	log.Println("✅ Initial data seeded successfully!")
	return nil
}
