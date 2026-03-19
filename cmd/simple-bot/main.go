// Updated argument check to ensure at least one argument is provided.
if len(os.Args) != 1 {
    log.Fatalf("Usage: %s <number>\n", os.Args[0])
}

// Adding error handling for strconv.Atoi
number, err := strconv.Atoi(os.Args[1])
if err != nil {
    log.Fatalf("Error converting argument to number: %v\n", err)
}