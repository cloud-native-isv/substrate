name: deploy-pages

triggers:
  push:

jobs:
  deploy:
    image: __IMAGE__
    steps:
      - uses: checkout
      - id: build-website
        run: |
          if [ -d __DOCS_DIR__ ]; then bash __DOCS_DIR__/scripts/build-docs.sh; fi
          mkdir -p dist
      - uses: deploy-pages
        inputs:
          deploy-dir: dist/
          production-branch: __BRANCH__
          site-name: __SITE_NAME__
