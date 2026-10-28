void main() {
  group('tracking group', () {
    void verifyTrackingBehavior({
      required String expectedSource,
      required String buttonFrom,
    }) {
      verify(() => repo.setVisitSource(source: expectedSource)).called(1);
      verify(() => mixpanel.trackEvent(eventName: 'x', properties: any(named: 'properties'))).called(1);
    }

    testWidgets('tracks button click', (tester) async {
      verifyTrackingBehavior(
        expectedSource: 'home',
        buttonFrom: 'submit_button',
      );
    });
  });
}
