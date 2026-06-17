import { Component, type ReactNode } from "react";

type ChartErrorBoundaryProps = {
  children: ReactNode;
  failedLabel?: string;
  resetKey: string;
};

type ChartErrorBoundaryState = {
  hasError: boolean;
};

export default class ChartErrorBoundary extends Component<
  ChartErrorBoundaryProps,
  ChartErrorBoundaryState
> {
  state: ChartErrorBoundaryState = {
    hasError: false,
  };

  static getDerivedStateFromError(): ChartErrorBoundaryState {
    return { hasError: true };
  }

  componentDidUpdate(previousProps: ChartErrorBoundaryProps) {
    if (previousProps.resetKey !== this.props.resetKey && this.state.hasError) {
      this.setState({ hasError: false });
    }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div
          className="empty-panel h-72 min-h-72"
          role="status"
        >
          {this.props.failedLabel ?? "차트를 불러오지 못했습니다."}
        </div>
      );
    }

    return this.props.children;
  }
}
