
db = db.getSiblingDB('migration_examples');

db.users.insertMany([
  {
    _id: ObjectId(),
    email: "JOHN.DOE@EXAMPLE.COM",
    first_name: "John",
    last_name: "Doe",
    status: "active",
    created_at: new Date("2024-01-01T10:00:00Z")
  },
  {
    _id: ObjectId(),
    email: "jane.smith@EXAMPLE.COM",
    first_name: "Jane",
    last_name: "Smith",
    status: "inactive",
    created_at: new Date("2024-01-02T14:30:00Z")
  },
  {
    _id: ObjectId(),
    email: "bob.johnson@example.com",
    first_name: "Bob",
    last_name: "Johnson",
    status: "active",
    created_at: new Date("2024-01-03T09:15:00Z"),
    updated_at: new Date("2024-01-03T09:15:00Z")
  },
  {
    _id: ObjectId(),
    email: "ALICE.WILLIAMS@EXAMPLE.COM",
    first_name: "Alice",
    status: "pending",
    created_at: new Date("2024-01-04T16:45:00Z")
  },
  {
    _id: ObjectId(),
    email: "charlie.brown@example.com",
    // first_name missing
    last_name: "Brown",
    status: "active",
    created_at: new Date("2024-01-05T11:20:00Z")
  }
]);

print("✅ Created migration_examples database with sample users");
print("📊 Inserted " + db.users.countDocuments({}) + " sample users");

print("\n📋 Sample data overview:");
db.users.find({}, {email: 1, first_name: 1, last_name: 1, status: 1}).forEach(printjson);

print("\nReady for migration examples!");
