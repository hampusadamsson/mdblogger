# Example

Run the following to host the blog from the path Arg

```
go run main.go /home/user/blog/content
```

# Configuration

```
/
|assets/
|-cat.png
|-dog.png
|post1.md
|post2.md

```

All links to internal assets from \*.md has to be placed in assets using the Obsidian style linking:

```
[[cat.png]]
```

# About section

The about/ link will simply link to an about page inside your path (argument starting the cli). It has to be called "about.md" and behave exactly like a normal post.
