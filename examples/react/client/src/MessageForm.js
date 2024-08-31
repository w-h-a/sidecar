import React from 'react';

export class MessageForm extends React.Component {
  constructor(props) {
    super(props);

    this.state = this.getInitialState();

    this.handleInputChange = this.handleInputChange.bind(this);

    this.handleSubmit = this.handleSubmit.bind(this);
  }

  getInitialState() {
    return {
      topic: 'arn:aws:sns:us-west-2:339936612855:neworder',
      message: `{}`,
    };
  }

  handleInputChange(event) {
    const target = event.target;

    const value = target.value;

    const name = target.name;

    console.log(`setting ${name} to ${value}`);

    this.setState({
      [name]: value,
    });
  }

  async handleSubmit(event) {
    event.preventDefault();

    await fetch('/publish', {
      headers: {
        accept: 'application/json',
        'content-type': 'application/json',
      },
      method: 'POST',
      body: JSON.stringify({
        eventName: this.state.topic,
        data: JSON.parse(this.state.message),
        to: [this.state.topic],
      }),
    });

    this.setState(this.getInitialState());
  }

  render() {
    return (
      <div class="col-12 col-md-9 col-xl-8 py-md-3 pl-md-5 bd-content">
        <form onSubmit={this.handleSubmit}>
          <div className="form-group">
            <label>Select Message Topic</label>
            <select
              value={this.state.topic}
              className="custom-select custom-select-lg mb-3"
              name="topic"
              onChange={this.handleInputChange}
            >
              <option value="arn:aws:sns:us-west-2:339936612855:neworder">
                neworder
              </option>
              <option value="arn:aws:sns:us-west-2:339936612855:b">b</option>
            </select>
          </div>
          <div className="form-group">
            <label>Enter JSON message</label>
            <textarea
              className="form-control"
              id="exampleFormControlTextarea1"
              rows="3"
              name="message"
              onChange={this.handleInputChange}
              value={this.state.message}
              placeholder="Enter JSON message here"
            ></textarea>
          </div>
          <button type="submit" className="btn btn-primary">
            Submit
          </button>
        </form>
      </div>
    );
  }
}
