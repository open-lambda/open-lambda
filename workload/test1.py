def lambda_handler(event, context):
    awesome_people = ["Andrea", "Remzi", "Scott", "Stephen", "Tyler"]
    if event["Name"] in awesome_people:
        return "You're Awesome!"
    return "Hello, " + event["Name"]
