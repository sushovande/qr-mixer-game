# Setting up your questions.

## Importing the list of players

Now, visit your site /9283e316-beaa-4182-b3a6-0937046251ee/manageUsers. Bookmark that page for ease of use. Follow the instructions on that page to import the names of all the attendees of your game. That URL is intentionally made long and obscure.

## Setting up the survey questions
Now, switch to the questions tab. There, you can set up all the questions for your game. In a typical game, there are 19 questions of varying difficulty.

The first box on the page is where you can setup the survey for the players. A survey is an opportunity for you to collect some information from the players that can be converted into a game question. Each survey question has an ID, and you can reference the survey question in your game question below. For example, if your survey question is 'Do you like the color blue?', you can then set a game question like, 'Find someone who is pleased by the color of the sky or the ocean'. Think of 5-6 survey questions and clues pointing to them. For each survey question, add a paragraph like this in the Survey Questions box:

```
survey_questions: {
  question_id: <some_number>
  question_text: "<question text>"
  type: BOOLEAN
}
```

The question id should be a number, starting from 1 for the first question. There can be no duplicates. This ID can be referenced by a game question later on.

The question text should be a straightforward question for the users to answer with a Yes or a No. You can put any HTML you like here as well, there are no restrictions. Usually, plain text works best.

The type must be BOOLEAN. There are no other types implemented right now.

## Setting up the game questions
Now start composing game questions with interesting or challenging questions about the players. Remember: you have to come up with 19 questions total.

For each question, you have to add a paragraph like this to the game questions box. Do not omit the curly braces and double quotes, they are important:

```
game_questions: {
  question_id: <some number>
  type: USERNAME_LIST / SURVEY_ANS / ANY_PERSON
  question_html: "<question text>"
  ans_usernames: "<first answer option>"
  ans_usernames: "<second answer option>"
  survey_id: <a number referring to the survey question. only if the type is SURVEY_ANS.>
  survey_true_is_correct: <true / false, only if the type is SURVEY_ANS>
}
```

The question id should be a number. The first question must be 1, the second question must be numbered 2, and so on. There can be no gaps, and no duplicates.

If your question is asking about a particular person, set the type to USERNAME_LIST, and include only a single ans_usernames line with that person's username in it.

If your question is asking about a group of people, set the type to USERNAME_LIST, and include multiple lines of ans_usernames, one for each username that should be accepted as an answer to your question. There is no limit to the number of players you can designate as an answer.

If your question is asking about some prop in the room, again set the type to USERNAME_LIST, and include only a single ans_usernames line with the username allocated to that prop. This username will typically start with the word "zspare".

If your question is referencing a survey question, set the type to SURVEY_ANS. Do not include any ans_usernames lines. Add a line for the survey_id, referencing the id of the question from the Survey Questions section. If you want to target those players who answered "Yes" to the survey, then add a line with survey_true_is_correct set to true. If you want to target those who answered "No" instead, then add the line survey_true_is_correct set to false.

If your question should allow all players to be scanned as a correct answer, then set the type to ANY_PERSON, and do not set any ans_usernames, survey_id, or survey_true_is_correct lines.

The question html field should have the text of your question. It is a good idea to preface every question with the question number, just so people know where they are in the game. You can use any HTML you like in this field, there are no restrictions. It is best not to go too crazy with the HTML, though.

Once you have figured out all the clues, shuffle them in some order, and preferably create a story line tying them all together. Enter all your questions on the questions page of the admin interface in the format provided.

Note: in the current version of the game, the following questions have special properties:
 * Questions 7 and 8: Answering any of these correctly has a chance to grant the player the first of four tokens.
 * Question 9: If the user hasn't received a token by this time, they are guaranteed to get a token by solving this question.
 * Questions 17 and 18: Answering any of these correctly has a chance to grant the player the second of four tokens.
 * Question 19: This is the last "regular" question. Answering this correctly guarantees that the player will receive the second of four tokens, in case they didn't get one already
 * Question 20: In this question, we expect the players to mingle among themselves, and share the tokens among each other until they get all four tokens.
 * Question 21: This is the final question. Here, the players do one final scan to finish the game.

## Navigation
 * Previous page: [Setting up the software](setting-up.md)
 * Next page: [Tips for making good questions](question-tips.md)