# gungus

A discord bot made for the miscellaneous needs of mine and my friends' server. Intended to be self-hosted.
Docker image is provided at `ghcr.io/lebulldoge/gungus`

### Usage:
```sh
$ gungus -token <your discord app token>
```
### Docker:
```sh
$ docker run -v <path to storage directory>:/config ghcr.io/lebulldoge/gungus -token <your discord app token>
```
### Current functionality:
* User polling

Command `/poll start` generates a poll with up to 6 options, 2 of which are required:

```
/poll start title: Favorite Chip option_0: 🔥;Sweet Chili Heat Doritos option_1: 🧀;Chili Cheese Fritos option_2: 🧂;Salt & Vinegar Pringles
```
![starting poll](https://github.com/LeBulldoge/gungus/assets/13983982/1cc215a4-b501-4746-9fd9-70deb1583d0b)

* Quotes

Command `/quote add` saves a quote by a particular user, `/quote random` to show a random quote:
```
/quote add by_user:@Gungus text:This is a quote
/quote random by_user:@Gungus
```

* Movie list

`/movie list` show the list of movies.
`/movie add` adds a movie to a list of watched movies in the guild. Provides autocompletion to select a movie from imdb.
`/movie cast` lets a user tag yourself as someone from the movie.
`/movie rate` rate a movie on a scale from -10.0 to 10.0. (where -10 is so bad it's good)
`/movie remove` remove a movie from the list.
```
/movie add title:Face/Off
/movie rate title:Face/Off rating:-6.0
/movie cast title:Face/Off character:Castor Troy
```

* Youtube audio playback

`/play` adds a link to the playback queue.
`/stop` stops playback and disconnects the bot.
