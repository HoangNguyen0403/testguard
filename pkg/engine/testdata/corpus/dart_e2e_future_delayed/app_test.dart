import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('integration app flow with hard delay', (tester) async {
    await Future.delayed(const Duration(seconds: 2));
    expect(find.text('Ready'), findsOneWidget);
  });
}
