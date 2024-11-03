# HTML API

A simple REST API that returns the HTML content of a given URL in JSON format.

## Usage

Send a request to the root URL with the following parameters:
* `url`: The URL of the page you want to get the HTML content of.
* `selector`: A CSS selector to filter the content of the page.
* `raw`: If set to `true`, the raw HTML content will be returned instead of the text content.

The response will be a JSON object with the following fields:
* `url`: The URL of the page.
* `selector`: The CSS selector used to filter the content.
* `elements`: An array of objects representing the elements that match the selector.
