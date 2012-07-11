=======================================
 Twackup -- Back up your tweets easily
=======================================

Builds like any other Go package.

Runs like::

  twackup USERNAME DIRECTORY

``twackup`` will write tweets, as JSON, under the given directory. The
file name will be the tweet id.

Downloading is fully incremental; ``twackup`` will look for both newer
and older tweets than the ones the directory already has. Note that
this won't re-download tweets in the middle, if they are deleted from
the directory.
